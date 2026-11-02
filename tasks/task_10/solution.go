package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
	"uuid"
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
		tasks: make(map[string]Task),
	}
}
func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	trimmedTitle := strings.TrimSpace(title)
	if trimmedTitle == "" {
		return Task{}, ErrInvalidTitle
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	newTask := Task{
		ID:        uuid.New().String(),
		Title:     trimmedTitle,
		Done:      false,
		UpdatedAt: r.clock.Now(),
	}
	r.tasks[newTask.ID] = newTask
	return newTask, nil
}
func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, exists := r.tasks[id]
	return t, exists
}
func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()
	results := make([]Task, 0, len(r.tasks))
	for _, t := range r.tasks {
		if t.Done == done {
			results = append(results, t)
		}
	}
	slices.SortFunc(results, func(a, b Task) int {
		if a.UpdatedAt.Compare(b.UpdatedAt) == 0 {
			return strings.Compare(a.ID, b.ID)
		}
		return b.UpdatedAt.Compare(a.UpdatedAt)
	})
	return results
}
func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task, exists := r.tasks[id]
	if !exists {
		return Task{}, ErrNotFound
	}
	if task.Done != done {
		task.Done = done
		task.UpdatedAt = r.clock.Now()
	}
	r.tasks[id] = task
	return task, nil
}

type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{repo: repo}
}
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	if url == "/tasks" && r.Method == http.MethodPost {
		h.handleCreate(w, r)
		return
	}
	if url == "/tasks" && r.Method == http.MethodGet {
		h.handleList(w, r)
		return
	}
	if url == "/tasks" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if id, ok := strings.CutPrefix(url, "/tasks/"); ok {
		if id == "" || strings.Contains(id, "/") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method == http.MethodGet {
			h.handleGet(w, r, id)
			return
		}
		if r.Method == http.MethodPatch {
			h.handlePatch(w, r, id)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.Error(w, "not found", http.StatusNotFound)
}
func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		Title *string `json:"title"`
	}
	if err := decodeStrictJSON(r.Body, &requestBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if requestBody.Title == nil {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}
	createdTask, err := h.repo.Create(*requestBody.Title)
	if err != nil {
		if errors.Is(err, ErrInvalidTitle) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, createdTask)
}
func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	task, exists := h.repo.Get(id)
	if !exists {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, task)
}
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	vals := query["done"]
	if len(vals) > 1 {
		http.Error(w, "query too many values", http.StatusBadRequest)
		return
	}
	var done bool
	if len(vals) == 1 {
		switch vals[0] {
		case "true":
			done = true
		case "false":
			done = false
		default:
			http.Error(w, "invalid query parameter", http.StatusBadRequest)
			return
		}
	}
	tasks := h.repo.List(done)
	writeJSON(w, http.StatusOK, tasks)
}
func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var requestBody struct {
		Done *bool `json:"done"`
	}
	if err := decodeStrictJSON(r.Body, &requestBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if requestBody.Done == nil {
		http.Error(w, "missing requiring args", http.StatusBadRequest)
		return
	}
	task, err := h.repo.SetDone(id, *requestBody.Done)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, task)
}
func decodeStrictJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(v)
	if err != nil {
		return err
	}
	var checkEmpty struct{}
	if err := decoder.Decode(&checkEmpty); err != io.EOF {
		return errors.New("unexpected data after JSON object")
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
