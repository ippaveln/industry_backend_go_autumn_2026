package main

import (
	"encoding/json"
	"errors"
	"fmt"
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
		tasks: make(map[string]Task),
	}
}
func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if len(title) == 0 {
		return Task{}, ErrInvalidTitle
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++
	task := Task{
		ID:        strconv.FormatUint(r.seq, 10),
		Title:     title,
		Done:      false,
		UpdatedAt: r.clock.Now(),
	}

	r.tasks[task.ID] = task

	return task, nil
}
func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[id]

	return task, exists
}
func (r *inMemoryTaskRepo) List(done bool) []Task {
	tasks := make([]Task, 0)

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, task := range r.tasks {
		if done == task.Done {
			tasks = append(tasks, task)
		}
	}

	return tasks
}
func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, exists := r.tasks[id]
	if !exists {
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
	return &httpHandler{repo}
}
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(pathParts) < 1 || len(pathParts) > 2 || pathParts[0] != "tasks" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "invalid path"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		if len(pathParts) == 2 {
			h.handleGet(w, r, pathParts[1])
		} else if len(pathParts) == 1 {
			h.handleList(w, r)
		} else {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "invalid path"})
		}
	case http.MethodPost:
		if len(pathParts) != 1 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "invalid path"})
			return
		}
		h.handleCreate(w, r)
	case http.MethodPatch:
		if len(pathParts) != 2 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "invalid path"})
			return
		}
		h.handlePatch(w, r, pathParts[1])
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "invalid method"})
	}
}
func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	type tempBodyStruct struct {
		Title *string `json:"title"`
	}
	var body tempBodyStruct
	err := decodeStrictJSON(r.Body, &body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if body.Title == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing title"})
		return
	}

	task, err := h.repo.Create(*body.Title)
	if err != nil {
		if errors.Is(err, ErrInvalidTitle) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/tasks/%s", task.ID))
	writeJSON(w, http.StatusCreated, task)
	return
}
func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	if id == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "invalid id"})
	}

	task, exists := h.repo.Get(id)
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}
	writeJSON(w, http.StatusOK, task)
	return
}
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	vals := r.URL.Query()["done"]
	if len(vals) == 0 {
		vals = []string{"false"}
	}
	if len(vals) != 1 || (vals[0] != "true" && vals[0] != "false") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid query"})
		return
	}

	done := vals[0] == "true"
	list := h.repo.List(done)
	slices.SortFunc(list, func(a Task, b Task) int {
		if a.UpdatedAt.Equal(b.UpdatedAt) {
			return strings.Compare(a.ID, b.ID)
		}

		return b.UpdatedAt.Compare(a.UpdatedAt)
	})
	writeJSON(w, http.StatusOK, list)
	return
}
func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	type tempBodyStruct struct {
		Done *bool `json:"done"`
	}
	var body tempBodyStruct
	err := decodeStrictJSON(r.Body, &body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if body.Done == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing done"})
		return
	}

	task, err := h.repo.SetDone(id, *body.Done)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, task)
	return
}
func decodeStrictJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(v)
	if err != nil {
		return fmt.Errorf("failed to unmarshal data: %v", err)
	}

	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("unexpected data after JSON object")
	}

	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
