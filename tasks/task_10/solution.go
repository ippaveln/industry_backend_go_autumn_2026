package main

import (
	"errors"
	"io"
	"net/http"
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
	panic("TODO: implement autumn contract")
}
func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	panic("TODO: implement autumn contract")
}
func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	panic("TODO: implement autumn contract")
}
func (r *inMemoryTaskRepo) List(done bool) []Task {
	panic("TODO: implement autumn contract")
}
func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	panic("TODO: implement autumn contract")
}

type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	panic("TODO: implement autumn contract")
}
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	panic("TODO: implement autumn contract")
}
func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	panic("TODO: implement autumn contract")
}
func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	panic("TODO: implement autumn contract")
}
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	panic("TODO: implement autumn contract")
}
func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	panic("TODO: implement autumn contract")
}
func decodeStrictJSON(r io.Reader, v any) error {
	panic("TODO: implement autumn contract")
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	panic("TODO: implement autumn contract")
}
