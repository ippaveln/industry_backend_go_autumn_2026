package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
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

// Глобальный счётчик обеспечивает уникальность ID
// даже при создании нескольких репозиториев.
var globalSeq atomic.Uint64

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

	id := strconv.FormatUint(globalSeq.Add(1), 10)

	task := Task{
		ID:        id,
		Title:     title,
		Done:      false,
		UpdatedAt: r.clock.Now(),
	}

	r.tasks[id] = task

	return task, nil
}

func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, ok := r.tasks[id]
	if !ok {
		return Task{}, false
	}

	return task, true
}

func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Task, 0)

	for _, task := range r.tasks {
		if task.Done == done {
			result = append(result, task)
		}
	}

	return result
}

func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}

	if task.Done != done {
		task.Done = done
		task.UpdatedAt = r.clock.Now()

		r.tasks[id] = task
	}

	return task, nil
}

type httpHandler struct {
	repo TaskRepo
}

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{
		repo: repo,
	}
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Коллекция задач.
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

	// Отдельная задача.
	if strings.HasPrefix(path, "/tasks/") {
		id := strings.TrimPrefix(path, "/tasks/")

		// Пустой ID или вложенный путь.
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
			w.Header().Set("Allow", "GET, PATCH")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}

		return
	}

	http.NotFound(w, r)
}

// POST /tasks
func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Title *string `json:"title"`
	}

	if err := decodeStrictJSON(r.Body, &request); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Title == nil {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	task, err := h.repo.Create(*request.Title)
	if err != nil {
		if errors.Is(err, ErrInvalidTitle) {
			http.Error(w, "invalid title", http.StatusBadRequest)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

// GET /tasks/{id}
func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	task, ok := h.repo.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// GET /tasks?done=false
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	done := false

	values, exists := r.URL.Query()["done"]

	if exists {
		if len(values) != 1 {
			http.Error(w, "invalid done parameter", http.StatusBadRequest)
			return
		}

		switch values[0] {
		case "true":
			done = true

		case "false":
			done = false

		default:
			http.Error(w, "invalid done parameter", http.StatusBadRequest)
			return
		}
	}

	tasks := h.repo.List(done)

	// Сортировка:
	// 1. UpdatedAt по убыванию.
	// 2. ID по возрастанию при одинаковом UpdatedAt.
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].UpdatedAt.Equal(tasks[j].UpdatedAt) {
			return tasks[i].ID < tasks[j].ID
		}

		return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
	})

	// Гарантируем [] вместо null.
	if tasks == nil {
		tasks = make([]Task, 0)
	}

	writeJSON(w, http.StatusOK, tasks)
}

// PATCH /tasks/{id}
func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var request struct {
		Done *bool `json:"done"`
	}

	if err := decodeStrictJSON(r.Body, &request); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Done == nil {
		http.Error(w, "done is required", http.StatusBadRequest)
		return
	}

	task, err := h.repo.SetDone(id, *request.Done)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// Строгий декодер JSON.

func decodeStrictJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()

	// Декодируем первый JSON-объект.
	if err := decoder.Decode(v); err != nil {
		return err
	}

	// Проверяем, что после объекта нет других JSON-значений.
	var extra json.RawMessage

	err := decoder.Decode(&extra)

	if err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}

		return err
	}

	return nil
}

// Запись JSON-ответа.

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(v)
}
