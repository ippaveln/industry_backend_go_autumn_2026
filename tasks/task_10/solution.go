package main

import (
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
	return &inMemoryTaskRepo{
		clock: clock,
		tasks: map[string]Task{},
	}

}
func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	formatTitle := strings.TrimSpace(title)

	if formatTitle == "" {
		return Task{}, ErrInvalidTitle
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++

	id := strconv.FormatUint(r.seq, 10)

	task := Task{
		ID:        id,
		Title:     formatTitle,
		Done:      false,
		UpdatedAt: r.clock.Now(),
	}

	r.tasks[id] = task

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

	result := make([]Task, 0, len(r.tasks))

	for _, task := range r.tasks {
		if task.Done == done {
			result = append(result, task)
		}
	}

	slices.SortFunc(result, func(a, b Task) int {
		if res := b.UpdatedAt.Compare(a.UpdatedAt); res != 0 {
			return res
		}
		return strings.Compare(a.ID, b.ID)
	})

	return result

}

func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]

	if !ok {
		return Task{}, ErrNotFound
	}

	if done == task.Done {
		return task, nil
	}

	task.Done = done
	task.UpdatedAt = r.clock.Now()
	r.tasks[id] = task

	return task, nil
}

type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{repo: repo}
}
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if path == "/tasks" {

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

	id, ok := strings.CutPrefix(path, "/tasks/")

	if !ok || id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, id)
	case http.MethodPatch:
		h.handlePatch(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Title *string `json:"title"`
	}

	if err := decodeStrictJSON(r.Body, &req); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if req.Title == nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	task, err := h.repo.Create(*req.Title)

	if err != nil {
		if errors.Is(err, ErrInvalidTitle) { // пустой title
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	task, ok := h.repo.Get(id)

	if !ok {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, task)
}
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	done := false

	values, present := r.URL.Query()["done"]

	if present {

		if len(values) != 1 {
			http.Error(w, "query parameter 'done' must be 1 value", http.StatusBadRequest)
			return
		}

		switch values[0] {
		case "true":
			done = true
		case "false":
			// пропускаем - он и так false
		default:
			http.Error(w, "query parameter 'done' must be either true or false", http.StatusBadRequest)
			return
		}
	}

	tasks := h.repo.List(done)
	writeJSON(w, http.StatusOK, tasks)
}
func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Done *bool `json:"done"`
	}

	if err := decodeStrictJSON(r.Body, &req); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if req.Done == nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	task, err := h.repo.SetDone(id, *req.Done)

	if err != nil {
		if errors.Is(err, ErrNotFound) { // пустой id
			http.NotFound(w, r)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, task)
}
func decodeStrictJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields() // неизвестные поля - ошибка

	if err := dec.Decode(v); err != nil {
		return err
	}

	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("body must contain a single JSON object")
	}

	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
