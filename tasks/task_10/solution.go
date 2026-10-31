package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List(done bool) []Task
	SetDone(id string, done bool) (Task, error)
}
type Clock interface{ Now() time.Time }

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("invalid title")
)

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	seq   uint64
	tasks map[string]Task
}

func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	repo := inMemoryTaskRepo{
		clock: clock,
		tasks: make(map[string]Task),
	}
	return &repo
}

func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	trimmedTitle := strings.TrimSpace(title)
	if trimmedTitle == "" {
		return Task{}, ErrInvalidTitle
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	stringID := strconv.Itoa(int(r.seq))
	newTask := Task{
		ID: stringID,
		Title: trimmedTitle,
		UpdatedAt: r.clock.Now(),
	}

	r.tasks[newTask.ID] = newTask
	r.seq ++
	return newTask, nil
}

func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, ok := r.tasks[id]
	return task, ok
}

func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	outputTasks := make([]Task, 0)
	
	for _, task := range r.tasks {
		if task.Done == done {
			outputTasks = append(outputTasks, task)
		}
	}
	
	return outputTasks
}

func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]

	if !ok {
		return Task{}, ErrNotFound
	}

	if task.Done == done {
		return task, nil
	}

	task.Done = done
	task.UpdatedAt = r.clock.Now()
	r.tasks[id] = task
	
	return task, nil 
}

type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	handler := httpHandler{repo: repo}
	return &handler
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/tasks" {
		switch r.Method {
		case http.MethodGet:
			h.handleList(w, r)
		case http.MethodPost:
			h.handleCreate(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Путь /tasks/{id}: id непустой и без вложенных сегментов.
	id, ok := strings.CutPrefix(r.URL.Path, "/tasks/")
	if ok && id != "" && !strings.Contains(id, "/") {
		switch r.Method {
		case http.MethodGet:
			h.handleGet(w, r, id)
		case http.MethodPatch:
			h.handlePatch(w, r, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	// Указатель отличает отсутствующее поле и null от обычного значения.
	var req struct {
		Title *string `json:"title"`
	}
	if err := decodeStrictJSON(r.Body, &req); err != nil || req.Title == nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	task, err := h.repo.Create(*req.Title)
	switch {
	case err == nil:
		writeJSON(w, http.StatusCreated, task)
	case errors.Is(err, ErrInvalidTitle):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	task, ok := h.repo.Get(id)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	// Без параметра done показываем незавершённые задачи.
	done := false
	if values, ok := r.URL.Query()["done"]; ok {
		if len(values) != 1 || (values[0] != "true" && values[0] != "false") {
			http.Error(w, "invalid done", http.StatusBadRequest)
			return
		}
		done = values[0] == "true"
	}

	tasks := h.repo.List(done)
	// nil-срез закодировался бы как null, а нужен [].
	if tasks == nil {
		tasks = []Task{}
	}
	// UpdatedAt по убыванию, при равенстве ID по возрастанию как строки.
	slices.SortFunc(tasks, func(a, b Task) int {
		return cmp.Or(
			b.UpdatedAt.Compare(a.UpdatedAt),
			cmp.Compare(a.ID, b.ID),
		)
	})
	writeJSON(w, http.StatusOK, tasks)
}

func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	// Указатель отличает отсутствующее поле и null от значения false.
	var req struct {
		Done *bool `json:"done"`
	}
	if err := decodeStrictJSON(r.Body, &req); err != nil || req.Done == nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	task, err := h.repo.SetDone(id, *req.Done)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, task)
	case errors.Is(err, ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func decodeStrictJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	// Первое чтение: сам объект.
	if err := dec.Decode(v); err != nil {
		return err
	}

	// Второе чтение: после объекта должен быть только конец данных.
	err := dec.Decode(&struct{}{})
	if errors.Is(err, io.EOF) {
		return nil
	}
	return errors.New("unexpected data after JSON object")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(body)
}
