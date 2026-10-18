package main

import (
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
	"sync/atomic"
	"strconv"
	"strings"
	"encoding/json"
	"sort"
)

//для id задачи сделала общий счетчик для всех репозиториев 
var taskIDCounter atomic.Uint64

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
		mu:    sync.RWMutex{},
		clock: clock,
		seq:   0,
		tasks: make(map[string]Task),
	}	
}
func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	// проверка на пустую строку
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrInvalidTitle
	}
	// блокировка на время создания
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++
	number := taskIDCounter.Add(1)
	task := Task{
		ID:        strconv.FormatUint(number, 10),
		Title:     title,
		Done:      false,
		UpdatedAt: r.clock.Now(),
	}
	r.tasks[task.ID] = task
	return task, nil
}
func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock() // блокировка для чтения
	defer r.mu.RUnlock()
	task, ok := r.tasks[id]
	return task, ok
}
func (r *inMemoryTaskRepo) List(done bool) []Task {
	newTasks := []Task{}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, task := range r.tasks {
		if task.Done == done {
			newTasks = append(newTasks, task)
		}
	}
	return newTasks
}
func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task, ok := r.tasks[id]
	// проверка существования задачи
	if !ok {
		return Task{}, ErrNotFound
	}
	// проверка на совпадение статуса задачи
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
path := r.URL.Path

	if path == "/tasks" {
		switch r.Method {
		case http.MethodGet:
			h.handleList(w, r)
		case http.MethodPost:
			h.handleCreate(w, r)
		default:
			// Неподдерживаемый метод на известном пути возвращаем 405
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if strings.HasPrefix(path, "/tasks/") {
		id := strings.TrimPrefix(path, "/tasks/")
		if id == "" || strings.Contains(id, "/") {
			// пустой id или вложенный путь - 404
			http.NotFound(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.handleGet(w, r, id)
		case http.MethodPatch:
			h.handlePatch(w, r, id)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	http.NotFound(w, r)
}
func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	// описываю тело запроса
	var req struct {
		Title *string `json:"title"`
	}
	err := decodeStrictJSON(r.Body, &req)
	if err != nil {
		// если ошибка декодирования, возвращаем 400
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if req.Title == nil {
		// если заголовок не передан, возвращаем 400
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	task, err := h.repo.Create(*req.Title)
	if err != nil {
		if errors.Is(err, ErrInvalidTitle) {
			// если что-то не так с заголовком, возвращаем 400
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		// в остальных случаях возвращаем 500
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
	// получаем значение done из заголовка
	values, exists := r.URL.Query()["done"]

	//отсутствие done равнозначно false, поэтому по умолчанию done = false
	done := false 

	if exists {
		if len(values) != 1 || (values[0] != "true" && values[0] != "false") {
			// если done не true и не false, возвращаем 400
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		done = values[0] == "true"
	}

	tasks := h.repo.List(done)
	if tasks == nil {
		tasks = []Task{}
	}
	//сортировка по времени
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].UpdatedAt.Equal(tasks[j].UpdatedAt) {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
	})
	writeJSON(w, http.StatusOK, tasks)
}

func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Done *bool `json:"done"`
	}

	err := decodeStrictJSON(r.Body, &req)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if req.Done == nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	task, err := h.repo.SetDone(id, *req.Done)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, task)
}
func decodeStrictJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields() // запрещаем неизвестные поля
	// разбираем JSON в переданную структуру v
	if err := decoder.Decode(v); err != nil {
		return err
	}

	// пытаемся прочитать дополнительные значения
	var extra any
	err := decoder.Decode(&extra)
	if err == io.EOF {
		// если ничего не прочиталось, то все хорошо
		return nil
	}
	if err != nil {
		return err
	}
	return errors.New("unexpected data after JSON")
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
