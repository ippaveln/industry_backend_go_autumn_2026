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
	id := strconv.FormatUint(r.seq, 10)
	r.seq++

	t := Task{
		ID:        id,
		Title:     title,
		Done:      false,
		UpdatedAt: r.clock.Now(),
	}

	r.tasks[id] = t
	return t, nil
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

	res := make([]Task, 0)
	for _, t := range r.tasks {
		if t.Done == done {
			res = append(res, t)
		}
	}
	return res
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

type httpHandler struct {
	repo TaskRepo
	mux  *http.ServeMux
}

type createTaskRequest struct {
	Title *string `json:"title"`
}

type patchTaskRequest struct {
	Done *bool `json:"done"`
}

func NewHTTPHandler(repo TaskRepo) http.Handler {
	h := &httpHandler{
		repo: repo,
		mux:  http.NewServeMux(),
	}
	h.mux.HandleFunc("POST /tasks", h.handleCreate)
	h.mux.HandleFunc("GET /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		h.handleGet(w, r, r.PathValue("id"))
	})
	h.mux.HandleFunc("GET /tasks", h.handleList)
	h.mux.HandleFunc("PATCH /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		h.handlePatch(w, r, r.PathValue("id"))
	})
	return h
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := decodeStrictJSON(r.Body, &req); err != nil || req.Title == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	task, err := h.repo.Create(*req.Title)
	switch {
	case err == nil:
		writeJSON(w, http.StatusCreated, task)
	case errors.Is(err, ErrInvalidTitle):
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	task, ok := h.repo.Get(id)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	done := false
	if values, ok := r.URL.Query()["done"]; ok {
		if len(values) != 1 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch values[0] {
		case "true":
			done = true
		case "false":
			done = false
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	tasks := h.repo.List(done)
	if tasks == nil {
		tasks = make([]Task, 0)
	}

	slices.SortFunc(tasks, func(a, b Task) int {
		if c := b.UpdatedAt.Compare(a.UpdatedAt); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})

	writeJSON(w, http.StatusOK, tasks)
}

func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var req patchTaskRequest
	if err := decodeStrictJSON(r.Body, &req); err != nil || req.Done == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	task, err := h.repo.SetDone(id, *req.Done)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, task)
	case errors.Is(err, ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func decodeStrictJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}

	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return errors.New("body must contain exactly one JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
