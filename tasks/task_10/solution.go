package main

import (
	"cmp"
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
	task := Task{
		ID:        strconv.FormatUint(r.seq, 10),
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

	out := []Task{}
	for _, task := range r.tasks {
		if task.Done == done {
			out = append(out, task)
		}
	}
	return out
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

const maxBody = 1 << 20

type httpHandler struct {
	repo TaskRepo
	mux  *http.ServeMux
}

func NewHTTPHandler(repo TaskRepo) http.Handler {
	h := &httpHandler{repo: repo, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /tasks", h.handleList)
	h.mux.HandleFunc("POST /tasks", h.handleCreate)
	h.mux.HandleFunc("GET /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		h.handleGet(w, r, r.PathValue("id"))
	})
	h.mux.HandleFunc("PATCH /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		h.handlePatch(w, r, r.PathValue("id"))
	})
	return h
}
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}
func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title *string `json:"title"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	if err := decodeStrictJSON(r.Body, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if req.Title == nil {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	task, err := h.repo.Create(*req.Title)
	switch {
	case errors.Is(err, ErrInvalidTitle):
		writeError(w, http.StatusBadRequest, err.Error())
	case err != nil:
		writeError(w, http.StatusInternalServerError, "internal error")
	default:
		writeJSON(w, http.StatusCreated, task)
	}
}
func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	task, ok := h.repo.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, ErrNotFound.Error())
		return
	}
	writeJSON(w, http.StatusOK, task)
}
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	done := false
	if values, ok := r.URL.Query()["done"]; ok {
		if len(values) != 1 {
			writeError(w, http.StatusBadRequest, "done must be given once")
			return
		}
		switch values[0] {
		case "true":
			done = true
		case "false":
		default:
			writeError(w, http.StatusBadRequest, "done must be true or false")
			return
		}
	}

	tasks := slices.Clone(h.repo.List(done))
	if tasks == nil {
		tasks = []Task{}
	}
	slices.SortFunc(tasks, func(a, b Task) int {
		if c := b.UpdatedAt.Compare(a.UpdatedAt); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	writeJSON(w, http.StatusOK, tasks)
}
func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Done *bool `json:"done"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	if err := decodeStrictJSON(r.Body, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if req.Done == nil {
		writeError(w, http.StatusBadRequest, "done is required")
		return
	}

	task, err := h.repo.SetDone(id, *req.Done)
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case err != nil:
		writeError(w, http.StatusInternalServerError, "internal error")
	default:
		writeJSON(w, http.StatusOK, task)
	}
}
func decodeStrictJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	if err := dec.Decode(&json.RawMessage{}); err != io.EOF {
		return errors.New("invalid json: unexpected data after object")
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeDecodeError(w http.ResponseWriter, err error) {
	if tooBig, ok := errors.AsType[*http.MaxBytesError](err); ok {
		writeError(w, http.StatusRequestEntityTooLarge, tooBig.Error())
		return
	}
	writeError(w, http.StatusBadRequest, err.Error())
}
