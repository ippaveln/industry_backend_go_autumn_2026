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

const (
	pathTasks           = "/tasks"
	pathTask            = "/tasks/"
	msgMethodNotAllowed = "method not allowed"
	msgBadDoneParam     = "invalid done parameter"
	msgDoneRequired     = "done is required"
	msgTitleRequired    = "title is required"
	contentTypeJSON     = "application/json"
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
	ErrNotFound	    = errors.New("task not found")
	ErrInvalidTitle = errors.New("invalid title")
)

var ErrMultipleJSON = errors.New("multiple JSON values in body")

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	seq   uint64
	tasks map[string]Task
}

type httpHandler struct{ repo TaskRepo }

type createTaskBody struct {
	Title *string `json:"title"`
}

type patchTaskBody struct {
	Done *bool `json:"done"`
}

func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	return &inMemoryTaskRepo{
		clock: clock,
		tasks: make(map[string]Task),
	}
}

func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrInvalidTitle
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++
	task := Task{
		ID:		strconv.FormatUint(r.seq, 10),
		Title:	 title,
		UpdatedAt: r.clock.Now(),
	}
	r.tasks[task.ID] = task
	return task, nil
}

func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.tasks[id]
	return t, ok
}

func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Task, 0, len(r.tasks))
	for _, t := range r.tasks {
		if t.Done == done {
			result = append(result, t)
		}
	}
	return result
}

func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	if t.Done == done {
		return t, nil
	}
	t.Done = done
	t.UpdatedAt = r.clock.Now()
	r.tasks[id] = t
	return t, nil
}

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{repo: repo}
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == pathTasks {
		switch r.Method {
		case http.MethodPost:
			h.handleCreate(w, r)
		case http.MethodGet:
			h.handleList(w, r)
		default:
			http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		}
		return
	}

	if id, ok := strings.CutPrefix(r.URL.Path, pathTask); ok && id != "" && !strings.Contains(id, "/") {
		switch r.Method {
		case http.MethodGet:
			h.handleGet(w, r, id)
		case http.MethodPatch:
			h.handlePatch(w, r, id)
		default:
			http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		}
		return
	}
	http.NotFound(w, r)
}

func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var body createTaskBody
	if err := decodeStrictJSON(r.Body, &body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.Title == nil {
		http.Error(w, msgTitleRequired, http.StatusBadRequest)
		return
	}

	task, err := h.repo.Create(*body.Title)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrInvalidTitle) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
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
	if vals, ok := r.URL.Query()["done"]; ok {
		if len(vals) != 1 {
			http.Error(w, msgBadDoneParam, http.StatusBadRequest)
			return
		}
		switch vals[0] {
		case "true":
			done = true
		case "false":
			done = false
		default:
			http.Error(w, msgBadDoneParam, http.StatusBadRequest)
			return
		}
	}

	tasks := h.repo.List(done)
	slices.SortFunc(tasks, func(a, b Task) int {
		if a.UpdatedAt.After(b.UpdatedAt) {
			return -1
		}
		if a.UpdatedAt.Before(b.UpdatedAt) {
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})

	writeJSON(w, http.StatusOK, tasks)
}

func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var body patchTaskBody
	if err := decodeStrictJSON(r.Body, &body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.Done == nil {
		http.Error(w, msgDoneRequired, http.StatusBadRequest)
		return
	}

	task, err := h.repo.SetDone(id, *body.Done)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func decodeStrictJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}

	var extra any
	err := dec.Decode(&extra)
	if err == nil {
		return ErrMultipleJSON
	}
	if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
