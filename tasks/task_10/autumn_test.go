package main

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestHTTP_List_DefaultsToPending(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"/tasks", "/tasks?done=false"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			h := handlerWithDoneTask(t)

			rr := do(t, h, http.MethodGet, path, nil)

			if rr.Code != http.StatusOK {
				t.Fatalf("GET %s: expected 200, got %d: %s", path, rr.Code, rr.Body.String())
			}
			if got := decodeJSON[[]Task](t, rr.Body); got == nil || len(got) != 0 {
				t.Fatalf("GET %s = %v, want empty JSON array []", path, got)
			}
		})
	}
}

func TestHTTP_List_DoneFilter(t *testing.T) {
	t.Parallel()

	h := handlerWithDoneTask(t)

	rr := do(t, h, http.MethodGet, "/tasks?done=true", nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := decodeJSON[[]Task](t, rr.Body); len(got) != 1 {
		t.Fatalf("expected 1 done task, got %d", len(got))
	}
}

func TestHTTP_List_InvalidDoneFilter_400(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		query string
	}{
		{name: "empty", query: ""},
		{name: "number", query: "1"},
		{name: "upper case", query: "TRUE"},
		{name: "repeated", query: "true&done=false"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := handlerWithDoneTask(t)

			rr := do(t, h, http.MethodGet, "/tasks?done="+tc.query, nil)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("done=%s: expected 400, got %d", tc.query, rr.Code)
			}
		})
	}
}

type failingRepo struct{ err error }

func (f failingRepo) Create(string) (Task, error)        { return Task{}, f.err }
func (f failingRepo) Get(id string) (Task, bool)         { return Task{ID: id}, true }
func (f failingRepo) List(bool) []Task                   { return []Task{} }
func (f failingRepo) SetDone(string, bool) (Task, error) { return Task{}, f.err }

func TestHTTP_Create_RepoErrors(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{name: "storage failure", err: errors.New("storage"), want: http.StatusInternalServerError},
		{name: "invalid title", err: ErrInvalidTitle, want: http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := NewHTTPHandler(failingRepo{tc.err})

			rr := do(t, h, http.MethodPost, "/tasks", []byte(`{"title":"x"}`))

			if rr.Code != tc.want {
				t.Fatalf("POST: got %d, want %d", rr.Code, tc.want)
			}
		})
	}
}

func TestHTTP_PatchDone_RepoErrors(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{name: "storage failure", err: errors.New("storage"), want: http.StatusInternalServerError},
		{name: "not found", err: ErrNotFound, want: http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := NewHTTPHandler(failingRepo{tc.err})

			rr := do(t, h, http.MethodPatch, "/tasks/id", []byte(`{"done":true}`))

			if rr.Code != tc.want {
				t.Fatalf("PATCH: got %d, want %d", rr.Code, tc.want)
			}
		})
	}
}

func handlerWithDoneTask(t *testing.T) http.Handler {
	t.Helper()
	repo := NewInMemoryTaskRepo(newFakeClock(time.Unix(0, 0)))
	task := mustCreate(t, repo, "a")
	if _, err := repo.SetDone(task.ID, true); err != nil {
		t.Fatalf("SetDone error: %v", err)
	}
	return NewHTTPHandler(repo)
}
