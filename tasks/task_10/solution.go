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
	m := make(map[string]Task)
	repo := inMemoryTaskRepo{
		clock: clock,
		tasks: m,
	}
	return &repo
}
func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if len(title) == 0 {
		return Task{}, ErrInvalidTitle
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	task := Task{
		Title:     title,
		ID:        strconv.FormatUint(r.seq, 10),
		UpdatedAt: r.clock.Now(),
	}
	r.seq++
	r.tasks[task.ID] = task
	return task, nil
}
func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, found := r.tasks[id]
	if !found {
		return Task{}, false
	}
	return r.tasks[id], true
}
func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	m := make([]Task, 0)
	for _, val := range r.tasks {
		if val.Done == done {
			m = append(m, val)
		}
	}
	r.mu.RUnlock()
	return m
}
func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	if r.tasks[id].Done == done {
		return r.tasks[id], nil
	} else {
		t.Done = done
		t.UpdatedAt = r.clock.Now()
		r.tasks[id] = t
	}
	return r.tasks[id], nil
}

type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{repo: repo}
}
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	way := r.URL.Path
	if way == "/tasks" {
		switch r.Method {
		case http.MethodPost:
			h.handleCreate(w, r)
		case http.MethodGet:
			h.handleList(w, r)
		default:
			http.Error(w, "method not allow", http.StatusMethodNotAllowed)
		}
		return
	}
	if strings.HasPrefix(way, "/tasks/") {
		id := strings.TrimPrefix(way, "/tasks/")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.handleGet(w, r, id)
		case http.MethodPatch:
			h.handlePatch(w, r, id)
		default:
			http.Error(w, "method not allow", http.StatusMethodNotAllowed)
		}
		return
	}
	http.NotFound(w, r)
}

type createRequest struct {
	Title string
}

func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	err := decodeStrictJSON(r.Body, &req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	task, e := h.repo.Create(req.Title)
	if e == ErrInvalidTitle {
		http.Error(w, "Empty request", http.StatusBadRequest)
		return
	} else if e != nil {
		http.Error(w, "Another error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, task)
}
func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	t, err := h.repo.Get(id)
	if err == false {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, t)
}
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	l := r.URL.Query()["done"]
	done := false
	if len(l) > 1 {
		http.Error(w, "Inсorrect request", http.StatusBadRequest)
		return
	}
	if len(l) == 1 {
		if l[0] == "true" {
			done = true
		} else if l[0] != "false" {
			http.Error(w, "Inkorrect request", http.StatusBadRequest)
			return
		}
	}
	task := h.repo.List(done)
	sort.Slice(task, func(i, j int) bool {
		if !task[i].UpdatedAt.Equal(task[j].UpdatedAt) {
			return task[i].UpdatedAt.After(task[j].UpdatedAt)
		}
		return task[i].ID < task[j].ID
	})
	writeJSON(w, http.StatusOK, task)
}

type patchRequest struct {
	Done *bool
}

func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var req patchRequest
	err := decodeStrictJSON(r.Body, &req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if req.Done == nil {
		http.Error(w, "Incorrect input", http.StatusBadRequest)
		return
	}
	task, e := h.repo.SetDone(id, *req.Done)
	if e == ErrNotFound {
		http.NotFound(w, r)
		return
	}
	if e != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, task)
}
func decodeStrictJSON(r io.Reader, v any) error {
	js := json.NewDecoder(r)
	js.DisallowUnknownFields()
	err := js.Decode(v)
	if err != nil {
		return err
	}
	var extra any
	err = js.Decode(&extra)
	if err == nil {
		return errors.New("unexpected data after JSON")
	}
	if err != io.EOF {
		return err
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
