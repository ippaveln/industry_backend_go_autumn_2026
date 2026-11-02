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
	return &inMemoryTaskRepo{clock: clock, tasks: make(map[string]Task)}
}

func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrInvalidTitle
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	t := Task{
		ID:        strconv.FormatUint(r.seq, 10),
		Title:     title,
		UpdatedAt: r.clock.Now(),
	}
	r.tasks[t.ID] = t
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
	out := make([]Task, 0, len(r.tasks))
	for _, t := range r.tasks {
		if t.Done == done {
			out = append(out, t)
		}
	}
	r.mu.RUnlock()
	sortTasks(out)
	return out
}

func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	if t.Done != done {
		t.Done = done
		t.UpdatedAt = r.clock.Now()
		r.tasks[id] = t
	}
	return t, nil
}

func sortTasks(ts []Task) {
	sort.Slice(ts, func(i, j int) bool {
		if !ts[i].UpdatedAt.Equal(ts[j].UpdatedAt) {
			return ts[i].UpdatedAt.After(ts[j].UpdatedAt)
		}
		return ts[i].ID < ts[j].ID
	})
}

type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{repo: repo}
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/tasks" {
		switch r.Method {
		case http.MethodPost:
			h.handleCreate(w, r)
		case http.MethodGet:
			h.handleList(w, r)
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	id, found := strings.CutPrefix(path, "/tasks/")
	if !found || id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, id)
	case http.MethodPatch:
		h.handlePatch(w, r, id)
	default:
		w.Header().Set("Allow", "GET, PATCH")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title *string `json:"title"`
	}
	if err := decodeStrictJSON(r.Body, &body); err != nil || body.Title == nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	t, err := h.repo.Create(*body.Title)
	switch {
	case errors.Is(err, ErrInvalidTitle):
		http.Error(w, "invalid title", http.StatusBadRequest)
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
	default:
		writeJSON(w, http.StatusCreated, t)
	}
}

func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	t, ok := h.repo.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	done := false
	if vals, present := r.URL.Query()["done"]; present {
		if len(vals) != 1 {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		switch vals[0] {
		case "true":
			done = true
		case "false":
		default:
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
	}
	src := h.repo.List(done)
	out := make([]Task, len(src))
	copy(out, src)
	sortTasks(out)
	writeJSON(w, http.StatusOK, out)
}

func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Done *bool `json:"done"`
	}
	if err := decodeStrictJSON(r.Body, &body); err != nil || body.Done == nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	t, err := h.repo.SetDone(id, *body.Done)
	switch {
	case errors.Is(err, ErrNotFound):
		http.NotFound(w, r)
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
	default:
		writeJSON(w, http.StatusOK, t)
	}
}

func decodeStrictJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return errors.New("unexpected data after JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
