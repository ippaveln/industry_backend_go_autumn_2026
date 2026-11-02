package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	taskSequence    atomic.Uint64
)

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	tasks map[string]Task
}

func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	return &inMemoryTaskRepo{clock: clock, tasks: make(map[string]Task)}
}

func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)

	if title == "" {
		return Task{}, ErrInvalidTitle
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	task := Task{
		ID:        strconv.FormatUint(taskSequence.Add(1), 10),
		Title:     title,
		UpdatedAt: r.clock.Now(),
	}

	r.tasks[task.ID] = task

	return task, nil
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

	tasks := make([]Task, 0)

	for _, task := range r.tasks {
		if task.Done == done {
			tasks = append(tasks, task)
		}
	}

	return tasks
}

func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]

	if !ok {
		return Task{}, ErrNotFound
	}

	if task.Done != done {
		task.Done = done
		task.UpdatedAt = r.clock.Now()
		r.tasks[id] = task
	}

	return task, nil
}

type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	h := &httpHandler{repo: repo}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /tasks", h.handleCreate)
	mux.HandleFunc("GET /tasks/{id}", h.handleGet)
	mux.HandleFunc("GET /tasks", h.handleList)
	mux.HandleFunc("PATCH /tasks/{id}", h.handlePatch)

	methodNotAllowed := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

	mux.HandleFunc("HEAD /tasks", methodNotAllowed)
	mux.HandleFunc("HEAD /tasks/{id}", methodNotAllowed)

	return mux
}

func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title *string `json:"title"`
	}

	if err := decodeStrictJSON(r.Body, &body); err != nil || body.Title == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	task, err := h.repo.Create(*body.Title)

	if errors.Is(err, ErrInvalidTitle) {
		w.WriteHeader(http.StatusBadRequest)
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		writeJSON(w, http.StatusCreated, task)
	}
}

func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	task, ok := h.repo.Get(r.PathValue("id"))

	if !ok {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	done := false

	if values, exists := r.URL.Query()["done"]; exists {
		if len(values) != 1 || (values[0] != "true" && values[0] != "false") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		done = values[0] == "true"
	}

	tasks := append([]Task{}, h.repo.List(done)...)

	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].UpdatedAt.Equal(tasks[j].UpdatedAt) {
			return tasks[i].ID < tasks[j].ID
		}

		return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
	})

	writeJSON(w, http.StatusOK, tasks)
}

func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Done *bool `json:"done"`
	}

	if err := decodeStrictJSON(r.Body, &body); err != nil || body.Done == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	task, err := h.repo.SetDone(r.PathValue("id"), *body.Done)

	if errors.Is(err, ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		writeJSON(w, http.StatusOK, task)
	}
}

func decodeStrictJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(v); err != nil {
		return err
	}

	var extra any

	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("expected exactly one JSON object")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(v)
}
