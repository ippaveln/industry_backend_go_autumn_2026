package main

import (
	"errors"
	"io"
	"net/http"
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
	panic("TODO: implement")
}
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	panic("TODO: implement")
}
func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	panic("TODO: implement")
}
func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	panic("TODO: implement")
}
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	panic("TODO: implement")
}
func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	panic("TODO: implement")
}
func decodeStrictJSON(r io.Reader, v any) error {
	panic("TODO: implement")
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	panic("TODO: implement")
}
