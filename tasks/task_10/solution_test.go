package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

var testStart = time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock(t time.Time) *fakeClock { return &fakeClock{t: t} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = t
}

func (c *fakeClock) Add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func TestRepo_CreateGet(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(testStart)
	repo := NewInMemoryTaskRepo(fc)

	created, err := repo.Create("hello")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	got, ok := repo.Get(created.ID)

	if created.ID == "" {
		t.Error("expected non-empty ID")
	}
	if created.Title != "hello" {
		t.Errorf("Title = %q, want %q", created.Title, "hello")
	}
	if created.Done {
		t.Error("expected Done=false by default")
	}
	if !created.UpdatedAt.Equal(fc.Now()) {
		t.Errorf("UpdatedAt = %v, want %v", created.UpdatedAt, fc.Now())
	}
	if !ok {
		t.Fatal("Get: expected ok=true")
	}
	if !sameTask(got, created) {
		t.Errorf("Get = %+v, want %+v", got, created)
	}
}

func TestRepo_Create_TrimsTitle(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryTaskRepo(newFakeClock(testStart))

	task, err := repo.Create(" a ")

	if err != nil || task.Title != "a" {
		t.Fatalf("Create(\" a \") = %+v, %v; want title %q", task, err, "a")
	}
}

func TestRepo_Create_RejectsBlankTitle(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryTaskRepo(newFakeClock(testStart))

	_, err := repo.Create(" ")

	if !errors.Is(err, ErrInvalidTitle) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidTitle)
	}
}

func TestRepo_Get_NotFound(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryTaskRepo(newFakeClock(testStart))

	_, ok := repo.Get("missing")

	if ok {
		t.Fatal("expected ok=false")
	}
}

func TestRepo_SetDone_UpdatesTime(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(testStart)
	repo := NewInMemoryTaskRepo(fc)
	task := mustCreate(t, repo, "x")
	fc.Add(5 * time.Second)

	updated, err := repo.SetDone(task.ID, true)

	if err != nil {
		t.Fatalf("SetDone error: %v", err)
	}
	if !updated.Done {
		t.Error("expected Done=true")
	}
	if !updated.UpdatedAt.Equal(fc.Now()) {
		t.Errorf("UpdatedAt = %v, want %v", updated.UpdatedAt, fc.Now())
	}
}

func TestRepo_SetDone_Idempotent(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(testStart)
	repo := NewInMemoryTaskRepo(fc)
	task := mustCreate(t, repo, "x")
	fc.Add(time.Second)
	updated, err := repo.SetDone(task.ID, true)
	if err != nil {
		t.Fatalf("SetDone error: %v", err)
	}
	fc.Add(time.Second)

	same, err := repo.SetDone(task.ID, true)

	if err != nil {
		t.Fatalf("SetDone error: %v", err)
	}
	if same != updated {
		t.Fatalf("repeated SetDone changed task: got %+v, want %+v", same, updated)
	}
}

func TestRepo_SetDone_NotFound(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryTaskRepo(newFakeClock(testStart))

	_, err := repo.SetDone("missing", true)

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrNotFound)
	}
}

func TestRepo_List_FiltersByDone(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryTaskRepo(newFakeClock(testStart))
	done := mustCreate(t, repo, "done")
	pending := mustCreate(t, repo, "pending")
	if _, err := repo.SetDone(done.ID, true); err != nil {
		t.Fatalf("SetDone error: %v", err)
	}

	doneList := repo.List(true)
	pendingList := repo.List(false)

	if len(doneList) != 1 || doneList[0].ID != done.ID {
		t.Errorf("List(true) = %+v, want only %s", doneList, done.ID)
	}
	if len(pendingList) != 1 || pendingList[0].ID != pending.ID {
		t.Errorf("List(false) = %+v, want only %s", pendingList, pending.ID)
	}
}

func TestRepo_List_ReturnsCopy(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryTaskRepo(newFakeClock(testStart))
	mustCreate(t, repo, "a")
	mustCreate(t, repo, "b")
	list := repo.List(false)
	if len(list) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(list))
	}
	id := list[0].ID
	want := list[0].Title

	list[0].Title = "hacked"

	got, ok := repo.Get(id)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got.Title != want {
		t.Fatalf("internal state changed through List result: Title = %q, want %q", got.Title, want)
	}
}

func TestRepo_ConcurrentAccess_NoPanicsAndConsistentLen(t *testing.T) {
	t.Parallel()

	const n = 200
	repo := NewInMemoryTaskRepo(newFakeClock(testStart))
	var wg sync.WaitGroup

	for range n {
		wg.Go(func() {
			task, err := repo.Create("t")
			if err != nil {
				t.Errorf("Create error: %v", err)
				return
			}
			_, _ = repo.Get(task.ID)
			_ = repo.List(false)
			_, _ = repo.SetDone(task.ID, true)
		})
	}
	wg.Wait()

	if got := len(repo.List(true)); got != n {
		t.Fatalf("expected %d done tasks, got %d", n, got)
	}
}

func TestHTTP_Create_201_AndBody(t *testing.T) {
	t.Parallel()

	fc, h := newTestHandler()

	rr := do(t, h, http.MethodPost, "/tasks", []byte(`{"title":"buy milk"}`))

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	task := decodeJSON[taskDTO](t, rr.Body)
	if task.ID == "" {
		t.Error("expected non-empty id")
	}
	if task.Title != "buy milk" {
		t.Errorf("title = %q, want %q", task.Title, "buy milk")
	}
	if task.Done {
		t.Error("expected done=false")
	}
	if !task.UpdatedAt.Equal(fc.Now()) {
		t.Errorf("updatedAt = %v, want %v", task.UpdatedAt, fc.Now())
	}
}

func TestHTTP_Create_Validation_400(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "bad json", body: `{"title":`},
		{name: "blank title", body: `{"title":"   "}`},
		{name: "unknown field", body: `{"title":"x","extra":1}`},
		{name: "null body", body: `null`},
		{name: "empty object", body: `{}`},
		{name: "null title", body: `{"title":null}`},
		{name: "second json value", body: `{"title":"x"} {}`},
		{name: "trailing garbage", body: `{"title":"x"} trailing`},
		{name: "array body", body: `[]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, h := newTestHandler()

			rr := do(t, h, http.MethodPost, "/tasks", []byte(tc.body))

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("body=%s: expected 400, got %d: %s", tc.body, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHTTP_Create_TrailingWhitespace_201_JSON(t *testing.T) {
	t.Parallel()

	_, h := newTestHandler()

	rr := do(t, h, http.MethodPost, "/tasks", []byte(`{"title":"x"} `))

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); !isJSONContentType(got) {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
}

func TestHTTP_GetByID_200(t *testing.T) {
	t.Parallel()

	_, h := newTestHandler()
	created := createTask(t, h, "x")

	rr := do(t, h, http.MethodGet, "/tasks/"+created.ID, nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := decodeJSON[taskDTO](t, rr.Body); got.ID != created.ID {
		t.Fatalf("id = %s, want %s", got.ID, created.ID)
	}
}

func TestHTTP_GetByID_404(t *testing.T) {
	t.Parallel()

	_, h := newTestHandler()

	rr := do(t, h, http.MethodGet, "/tasks/missing", nil)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHTTP_UnknownRoute_404(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"/missing", "/tasks/", "/tasks/x/y"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			_, h := newTestHandler()

			rr := do(t, h, http.MethodGet, path, nil)

			if rr.Code != http.StatusNotFound {
				t.Fatalf("GET %s: expected 404, got %d", path, rr.Code)
			}
		})
	}
}

func TestHTTP_MethodNotAllowed_405(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodDelete, path: "/tasks/{id}"},
		{method: http.MethodPut, path: "/tasks"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			t.Parallel()
			_, h := newTestHandler()
			created := createTask(t, h, "x")
			path := strings.ReplaceAll(tc.path, "{id}", created.ID)

			rr := do(t, h, tc.method, path, nil)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Fatalf("%s %s: expected 405, got %d", tc.method, path, rr.Code)
			}
		})
	}
}

func TestHTTP_PatchDone_200_UpdatesTime(t *testing.T) {
	t.Parallel()

	fc, h := newTestHandler()
	created := createTask(t, h, "x")
	fc.Add(10 * time.Second)

	rr := do(t, h, http.MethodPatch, "/tasks/"+created.ID, []byte(`{"done":true}`))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	updated := decodeJSON[taskDTO](t, rr.Body)
	if !updated.Done {
		t.Error("expected done=true")
	}
	if !updated.UpdatedAt.Equal(fc.Now()) {
		t.Errorf("updatedAt = %v, want %v", updated.UpdatedAt, fc.Now())
	}
}

func TestHTTP_PatchDone_Validation_400(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "wrong type", body: `{"done":"true"}`},
		{name: "missing field", body: `{}`},
		{name: "unknown field", body: `{"done":true,"x":1}`},
		{name: "bad json", body: `{"done":`},
		{name: "null body", body: `null`},
		{name: "null done", body: `{"done":null}`},
		{name: "second json value", body: `{"done":false} {}`},
		{name: "array body", body: `[]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, h := newTestHandler()
			created := createTask(t, h, "x")

			rr := do(t, h, http.MethodPatch, "/tasks/"+created.ID, []byte(tc.body))

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("body=%s: expected 400, got %d: %s", tc.body, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHTTP_PatchDone_404(t *testing.T) {
	t.Parallel()

	_, h := newTestHandler()

	rr := do(t, h, http.MethodPatch, "/tasks/missing", []byte(`{"done":true}`))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHTTP_List_SortedByUpdatedAtDesc(t *testing.T) {
	t.Parallel()

	fc, h := newTestHandler()
	t1 := createTask(t, h, "a")
	fc.Add(time.Second)
	t2 := createTask(t, h, "b")
	fc.Add(time.Second)
	t3 := createTask(t, h, "c")

	rr := do(t, h, http.MethodGet, "/tasks", nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	got := taskIDs(decodeJSON[[]taskDTO](t, rr.Body))
	want := []string{t3.ID, t2.ID, t1.ID}
	if !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestHTTP_List_TieBrokenByIDAsc(t *testing.T) {
	t.Parallel()

	fc, h := newTestHandler()
	createTask(t, h, "older")
	fc.Set(testStart.Add(time.Hour))
	tied := []string{
		createTask(t, h, "tie1").ID,
		createTask(t, h, "tie2").ID,
		createTask(t, h, "tie3").ID,
	}
	want := slices.Sorted(slices.Values(tied))

	rr := do(t, h, http.MethodGet, "/tasks", nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var got []string
	for _, task := range decodeJSON[[]taskDTO](t, rr.Body) {
		if task.UpdatedAt.Equal(fc.Now()) && slices.Contains(tied, task.ID) {
			got = append(got, task.ID)
		}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("tie order = %v, want ids ascending %v", got, want)
	}
}

type taskDTO struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newTestHandler() (*fakeClock, http.Handler) {
	fc := newFakeClock(testStart)
	return fc, NewHTTPHandler(NewInMemoryTaskRepo(fc))
}

func mustCreate(t *testing.T, repo TaskRepo, title string) Task {
	t.Helper()
	task, err := repo.Create(title)
	if err != nil {
		t.Fatalf("Create(%q) error: %v", title, err)
	}
	return task
}

func createTask(t *testing.T, h http.Handler, title string) taskDTO {
	t.Helper()
	body, err := json.Marshal(map[string]string{"title": title})
	if err != nil {
		t.Fatal(err)
	}
	rr := do(t, h, http.MethodPost, "/tasks", body)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %q: expected 201, got %d: %s", title, rr.Code, rr.Body.String())
	}
	return decodeJSON[taskDTO](t, rr.Body)
}

func do(t *testing.T, h http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func decodeJSON[T any](t *testing.T, r io.Reader) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	return v
}

func isJSONContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && mediaType == "application/json"
}

func sameTask(a, b Task) bool {
	return a.ID == b.ID && a.Title == b.Title && a.Done == b.Done && a.UpdatedAt.Equal(b.UpdatedAt)
}

func taskIDs(tasks []taskDTO) []string {
	ids := make([]string, len(tasks))
	for i, task := range tasks {
		ids[i] = task.ID
	}
	return ids
}
