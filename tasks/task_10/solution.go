package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
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
)

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	tasks map[string]Task
}

var taskSequence atomic.Uint64

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

	created := Task{
		ID:        strconv.FormatUint(taskSequence.Add(1), 10),
		Title:     title,
		UpdatedAt: r.clock.Now(),
	}
	r.tasks[created.ID] = created
	return created, nil
}
func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	task, found := r.tasks[id]
	return task, found
}
func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()
	matched := make([]Task, 0)
	for _, task := range r.tasks {
		if task.Done == done {
			matched = append(matched, task)
		}
	}
	return matched
}
func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task, found := r.tasks[id]
	if !found {
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
	return &httpHandler{repo: repo}
}
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 2 || parts[1] != "tasks" || len(parts) > 3 || (len(parts) == 3 && parts[2] == "") {
		http.NotFound(w, r)
		return
	}
	collection := len(parts) == 2
	switch {
	case collection && r.Method == http.MethodPost:
		h.handleCreate(w, r)
	case collection && r.Method == http.MethodGet:
		h.handleList(w, r)
	case !collection && r.Method == http.MethodGet:
		h.handleGet(w, r, parts[2])
	case !collection && r.Method == http.MethodPatch:
		h.handlePatch(w, r, parts[2])
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title *string `json:"title"`
	}
	if err := decodeStrictJSON(r.Body, &body); err != nil || body.Title == nil {
		http.Error(w, "title must be a string in a JSON object", http.StatusBadRequest)
		return
	}
	created, err := h.repo.Create(*body.Title)
	switch {
	case errors.Is(err, ErrInvalidTitle):
		http.Error(w, "invalid title", http.StatusBadRequest)
	case err != nil:
		http.Error(w, "could not create task", http.StatusInternalServerError)
	default:
		writeJSON(w, http.StatusCreated, created)
	}
}
func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	task, found := h.repo.Get(id)
	if !found {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, task)
}
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	filter, present := r.URL.Query()["done"]
	done := false
	if present {
		if len(filter) != 1 || (filter[0] != "true" && filter[0] != "false") {
			http.Error(w, "done must be true or false and appear once", http.StatusBadRequest)
			return
		}
		done = filter[0] == "true"
	}
	tasks := append([]Task{}, h.repo.List(done)...)
	slices.SortFunc(tasks, func(a, b Task) int {
		if order := b.UpdatedAt.Compare(a.UpdatedAt); order != 0 {
			return order
		}
		return strings.Compare(a.ID, b.ID)
	})
	writeJSON(w, http.StatusOK, tasks)
}
func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Done *bool `json:"done"`
	}
	if err := decodeStrictJSON(r.Body, &body); err != nil || body.Done == nil {
		http.Error(w, "done must be a boolean in a JSON object", http.StatusBadRequest)
		return
	}
	updated, err := h.repo.SetDone(id, *body.Done)
	switch {
	case errors.Is(err, ErrNotFound):
		http.NotFound(w, r)
	case err != nil:
		http.Error(w, "could not update task", http.StatusInternalServerError)
	default:
		writeJSON(w, http.StatusOK, updated)
	}
}
func decodeStrictJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	var object json.RawMessage
	if err := decoder.Decode(&object); err != nil {
		return err
	}
	if len(object) == 0 || object[0] != '{' {
		return errors.New("expected a JSON object")
	}
	fields := json.NewDecoder(bytes.NewReader(object))
	fields.DisallowUnknownFields()
	if err := fields.Decode(v); err != nil {
		return err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("unexpected data after JSON object")
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
