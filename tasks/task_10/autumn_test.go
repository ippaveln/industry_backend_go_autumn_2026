package main

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestFilteredListAndIdempotence(t *testing.T) {
	c := newFakeClock(time.Unix(0, 0))
	r := NewInMemoryTaskRepo(c)
	a, _ := r.Create(" a ")
	b, _ := r.Create("b")
	c.Add(time.Second)
	updated, _ := r.SetDone(a.ID, true)
	c.Add(time.Second)
	same, _ := r.SetDone(a.ID, true)
	if same != updated {
		t.Error("idempotent update changed task")
	}
	if x := r.List(true); len(x) != 1 || x[0].ID != a.ID {
		t.Error("done filter", x)
	}
	if x := r.List(false); len(x) != 1 || x[0].ID != b.ID {
		t.Error("pending filter", x)
	}
	if a.Title != "a" {
		t.Error("trim")
	}
	if _, e := r.Create(" "); !errors.Is(e, ErrInvalidTitle) {
		t.Error(e)
	}
	if _, e := r.SetDone("missing", true); !errors.Is(e, ErrNotFound) {
		t.Error(e)
	}
}
func TestHTTPFilterAndStrictJSON(t *testing.T) {
	r := NewInMemoryTaskRepo(newFakeClock(time.Unix(0, 0)))
	a, _ := r.Create("a")
	r.SetDone(a.ID, true)
	h := NewHTTPHandler(r)
	for _, path := range []string{"/tasks", "/tasks?done=false"} {
		rr := do(t, h, "GET", path, nil)
		if rr.Code != 200 || rr.Body.String() != "[]\n" {
			t.Error(path, rr.Code, rr.Body.String())
		}
	}
	rr := do(t, h, "GET", "/tasks?done=true", nil)
	if rr.Code != 200 || len(decodeJSON[[]Task](t, rr.Body)) != 1 {
		t.Fatal("completed filter")
	}
	for _, q := range []string{"", "1", "TRUE", "true&done=false"} {
		if rr := do(t, h, "GET", "/tasks?done="+q, nil); rr.Code != 400 {
			t.Error(q, rr.Code)
		}
	}
	for _, b := range []string{`null`, `{}`, `{"title":null}`, `{"title":"x"} {}`, `{"title":"x"} trailing`, `[]`} {
		if rr := do(t, h, "POST", "/tasks", []byte(b)); rr.Code != 400 {
			t.Error(b, rr.Code)
		}
	}
	for _, b := range []string{`null`, `{"done":null}`, `{"done":false} {}`, `[]`} {
		if rr := do(t, h, "PATCH", "/tasks/"+a.ID, []byte(b)); rr.Code != 400 {
			t.Error(b, rr.Code)
		}
	}
	for _, p := range []string{"/missing", "/tasks/", "/tasks/x/y"} {
		if rr := do(t, h, "GET", p, nil); rr.Code != 404 {
			t.Error(p, rr.Code)
		}
	}
	if rr := do(t, h, "DELETE", "/tasks/"+a.ID, nil); rr.Code != 405 {
		t.Error(rr.Code)
	}
	if rr := do(t, h, "POST", "/tasks", []byte(`{"title":"x"} `)); rr.Code != 201 || rr.Header().Get("Content-Type") != "application/json" {
		t.Error(rr.Code, rr.Header())
	}
}

type failingRepo struct{ err error }

func (f failingRepo) Create(string) (Task, error)        { return Task{}, f.err }
func (f failingRepo) Get(string) (Task, bool)            { return Task{}, false }
func (f failingRepo) List(bool) []Task                   { return []Task{} }
func (f failingRepo) SetDone(string, bool) (Task, error) { return Task{}, f.err }
func TestHTTPRepoInterfaceErrors(t *testing.T) {
	for _, c := range []struct {
		err         error
		post, patch int
	}{{errors.New("storage"), 500, 500}, {ErrInvalidTitle, 400, 500}, {ErrNotFound, 500, 404}} {
		h := NewHTTPHandler(failingRepo{c.err})
		if rr := do(t, h, http.MethodPost, "/tasks", []byte(`{"title":"x"}`)); rr.Code != c.post {
			t.Error(rr.Code, c.post)
		}
		if rr := do(t, h, http.MethodPatch, "/tasks/id", []byte(`{"done":true}`)); rr.Code != c.patch {
			t.Error(rr.Code, c.patch)
		}
	}
}
