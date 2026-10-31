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
	newRepo := inMemoryTaskRepo{
		clock: clock,
		tasks: make(map[string]Task),
	}
	return &newRepo
}
func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrInvalidTitle
	}
	id := strconv.FormatUint(r.seq, 10)

	newTask := Task{
		Title:     title,
		ID:        id,
		UpdatedAt: r.clock.Now(),
	}
	r.tasks[id] = newTask
	r.seq++
	return newTask, nil
}

func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	getTask, ok := r.tasks[id]
	return getTask, ok
}

func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()
	getList := make([]Task, 0)
	for _, t := range r.tasks {
		if t.Done == done {
			getList = append(getList, t)

		}
	}
	return getList
}

func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	getTask, ok := r.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	if getTask.Done == done {
		return getTask, nil
	}
	getTask.Done = done
	getTask.UpdatedAt = r.clock.Now()
	r.tasks[id] = getTask
	return getTask, nil
}

type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	h := &httpHandler{repo: repo}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks", h.handleCreate)
	mux.HandleFunc("GET /tasks", h.handleList)
	mux.HandleFunc("GET /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		h.handleGet(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("PATCH /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		h.handlePatch(w, r, r.PathValue("id"))
	})
	return mux
}
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	panic("TODO: implement")
}
func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title *string `json:"title"`
	}
	err := decodeStrictJSON(r.Body, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Title == nil {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}
	task, err := h.repo.Create(*req.Title)
	if err != nil {
		if errors.Is(err, ErrInvalidTitle) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	writeJSON(w, http.StatusCreated, task)
}
func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	taskGet, ok := h.repo.Get(id)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, taskGet)
}
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	done := false
	vals, ok := r.URL.Query()["done"]
	if ok {
		if len(vals) != 1 {
			http.Error(w, "done must be given once", http.StatusBadRequest)
			return
		}
		if vals[0] == "true" {
			done = true
		} else if vals[0] != "false" {
			http.Error(w, "done must be true or false", http.StatusBadRequest)
			return
		}
	}

	tasks := h.repo.List(done)
	if tasks == nil {
		tasks = []Task{}
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
	var req struct {
		Done *bool `json:"done"`
	}
	err := decodeStrictJSON(r.Body, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Done == nil {
		http.Error(w, "done is required", http.StatusBadRequest)
		return
	}
	task, err := h.repo.SetDone(id, *req.Done)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	writeJSON(w, http.StatusOK, task)
}
func decodeStrictJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	err := dec.Decode(v)
	if err != nil {
		return err
	}
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("extra data after JSON")
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
