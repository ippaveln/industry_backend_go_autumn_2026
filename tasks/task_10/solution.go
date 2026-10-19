package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
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
	r.mu.Lock()
	defer r.mu.Unlock()

	trimTitle := strings.TrimSpace(title)
	if trimTitle == "" {
		return Task{}, ErrInvalidTitle
	}
	newTask := Task{
		ID:        uuid.New().String(),
		Title:     trimTitle,
		Done:      false,
		UpdatedAt: r.clock.Now(),
	}
	r.tasks[newTask.ID] = newTask
	return newTask, nil
}

func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.tasks[id]
	if !ok {
		return Task{}, false
	}
	return t, true
}

func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Task, 0, len(r.tasks))
	for _, val := range r.tasks {
		if val.Done == done {
			result = append(result, val)
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
	if t.Done != done {
		t.Done = done
		t.UpdatedAt = r.clock.Now()
		r.tasks[id] = t
	}
	return t, nil
}

type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{
		repo: repo,
	}
}
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/tasks":
		switch r.Method {
		case http.MethodPost:
			h.handleCreate(w, r)
		case http.MethodGet:
			h.handleList(w, r)
		default:
			// RFC9110/#status.405
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

	case path.Dir(r.URL.Path) == "/tasks":
		id := path.Base(r.URL.Path)

		switch r.Method {
		case http.MethodGet:
			h.handleGet(w, r, id)
		case http.MethodPatch:
			h.handlePatch(w, r, id)
		default:
			// RFC9110/#status.405
			w.Header().Set("Allow", "GET, PATCH")
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

	default:
		http.Error(w, "tasks not found", http.StatusNotFound)
	}
}

func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	t := Task{}
	if err := decodeStrictJSON(r.Body, &t); err != nil {
		http.Error(w, "request body failed decode", http.StatusBadRequest)
		return
	}

	t, err := h.repo.Create(t.Title)
	if errors.Is(err, ErrInvalidTitle) {
		http.Error(w, "invalid title", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, 201, t)
}

func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	t, ok := h.repo.Get(id)
	if !ok {
		http.Error(w, "task id not found", http.StatusNotFound)
		return
	}
	writeJSON(w, 200, t)
}

func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		log.Printf("parse query error: %v", err)
		return
	}

	if len(q["done"]) > 1 {
		http.Error(w, "?done= incorrect in URL", http.StatusBadRequest)
		return
	}

	t := make([]Task, 0)
	switch {
	case !q.Has("done"):
		t = h.repo.List(false)
	case q.Get("done") == "true":
		t = h.repo.List(true)
	case q.Get("done") == "false":
		t = h.repo.List(false)
	default:
		http.Error(w, "?done= incorrect in URL", http.StatusBadRequest)
		return
	}

	slices.SortFunc(t, func(a, b Task) int {
		c := a.UpdatedAt.Compare(b.UpdatedAt)
		if c == 0 {
			return strings.Compare(a.ID, b.ID)
		}
		return c * -1
	})

	writeJSON(w, 200, t)
}

func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var t struct {
		Done *bool `json:"done"`
	}

	if err := decodeStrictJSON(r.Body, &t); err != nil {
		http.Error(w, "request body failed decode", http.StatusBadRequest)
		return
	}

	if t.Done == nil {
		http.Error(w, "invalid value status", http.StatusBadRequest)
		return
	}

	updTask, err := h.repo.SetDone(id, *t.Done)
	if errors.Is(err, ErrNotFound) {
		http.Error(w, "task id not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, 200, updTask)
}
func decodeStrictJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		log.Printf("json decode error: %v", err)
		return err
	}

	if t, err := dec.Token(); err != io.EOF {
		log.Printf("there is data after the request: %v", t)
		if err == nil {
			return errors.New("there is data after the request")
		}
		return err
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("serialization error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, err = w.Write(data)
	if err != nil {
		log.Printf("failed to write response: %v", err)
	}
}
