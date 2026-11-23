package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"testing/fstest"

	"unwind/internal/plan"
	"unwind/internal/store"
)

type memoryStore struct {
	mu       sync.Mutex
	progress map[string]plan.Progress
	feedback map[string]string
}

func newMemoryStore() *memoryStore {
	return &memoryStore{progress: make(map[string]plan.Progress), feedback: make(map[string]string)}
}
func (s *memoryStore) EnsureDevice(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.progress[id]; !ok {
		s.progress[id] = plan.Progress{DeviceID: id, Entries: []plan.Entry{}}
	}
	return nil
}
func (s *memoryStore) Progress(_ context.Context, id string) (plan.Progress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	progress := s.progress[id]
	progress.Entries = append([]plan.Entry(nil), progress.Entries...)
	return progress, nil
}
func (s *memoryStore) Complete(_ context.Context, id string, day int, mood, ritual string) (plan.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	progress := s.progress[id]
	for _, entry := range progress.Entries {
		if entry.Day == day {
			return entry, nil
		}
	}
	entry := plan.Entry{Day: day, Mood: mood, Ritual: ritual}
	progress.Entries = append(progress.Entries, entry)
	s.progress[id] = progress
	return entry, nil
}
func (s *memoryStore) Activate(_ context.Context, id, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if code != "VALID-CODE" {
		return store.ErrInvalidActivation
	}
	progress := s.progress[id]
	progress.Activated = true
	s.progress[id] = progress
	return nil
}
func (s *memoryStore) SaveFeedback(_ context.Context, id string, day int, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.feedback[fmt.Sprintf("%s:%d", id, day)] = text
	return nil
}

type apiGenerator struct {
	texts []string
}

func (g *apiGenerator) Generate(_ context.Context, _ int, text string) (plan.Reply, error) {
	g.texts = append(g.texts, text)
	return plan.Reply{Message: "今天辛苦了。", Action: "慢慢喝杯水。", Source: "ai"}, nil
}

type apiClient struct {
	t       *testing.T
	handler http.Handler
	cookie  *http.Cookie
}

func (c *apiClient) request(method, path, body string, wantStatus int) map[string]any {
	c.t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	if c.cookie != nil {
		request.AddCookie(c.cookie)
	}
	response := httptest.NewRecorder()
	c.handler.ServeHTTP(response, request)
	if response.Code != wantStatus {
		c.t.Fatalf("%s %s: status=%d want=%d body=%s", method, path, response.Code, wantStatus, response.Body.String())
	}
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			c.cookie = cookie
		}
	}
	var result map[string]any
	if response.Body.Len() > 0 {
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			c.t.Fatalf("invalid JSON response: %v", err)
		}
	}
	return result
}

func TestAPICompleteSevenDayFlow(t *testing.T) {
	store := newMemoryStore()
	generator := &apiGenerator{}
	client := &apiClient{t: t, handler: NewRouter(store, generator, fstest.MapFS{}, false)}

	status := client.request(http.MethodPost, "/api/session", "", http.StatusOK)
	if status["nextDay"] != float64(1) {
		t.Fatalf("unexpected initial status: %+v", status)
	}
	client.request(http.MethodPost, "/api/replies", `{"text":"今天开了很多会"}`, http.StatusOK)
	status = client.request(http.MethodPost, "/api/entries", `{"day":1,"mood":"平静","ritual":"揉掉"}`, http.StatusOK)
	if status["completedDays"] != float64(1) {
		t.Fatalf("unexpected first completion: %+v", status)
	}
	status = client.request(http.MethodPost, "/api/entries", `{"day":1,"mood":"平静","ritual":"揉掉"}`, http.StatusOK)
	if status["completedDays"] != float64(1) {
		t.Fatalf("repeat completion changed progress: %+v", status)
	}
	client.request(http.MethodPost, "/api/entries", `{"day":3,"mood":"平静","ritual":"揉掉"}`, http.StatusConflict)
	client.request(http.MethodPost, "/api/replies", `{"text":"第二天"}`, http.StatusOK)
	client.request(http.MethodPost, "/api/entries", `{"day":2,"mood":"疲惫","ritual":"封存"}`, http.StatusOK)
	client.request(http.MethodPost, "/api/replies", `{"text":"第三天"}`, http.StatusPaymentRequired)
	status = client.request(http.MethodPost, "/api/activate", `{"code":"VALID-CODE"}`, http.StatusOK)
	if status["activated"] != true {
		t.Fatalf("activation failed: %+v", status)
	}
	for day := 3; day <= 7; day++ {
		client.request(http.MethodPost, "/api/replies", fmt.Sprintf(`{"text":"第%d天"}`, day), http.StatusOK)
		client.request(http.MethodPost, "/api/entries", fmt.Sprintf(`{"day":%d,"mood":"平静","ritual":"揉掉"}`, day), http.StatusOK)
	}
	review := client.request(http.MethodGet, "/api/review", "", http.StatusOK)
	if review["completedDays"] != float64(7) {
		t.Fatalf("unexpected review: %+v", review)
	}
	if len(generator.texts) != 7 {
		t.Fatalf("generator calls = %d, want 7", len(generator.texts))
	}
	for id, progress := range store.progress {
		if len(id) != 64 {
			t.Fatalf("device ID is not a hash: %q", id)
		}
		for _, entry := range progress.Entries {
			if entry.Mood == "今天开了很多会" {
				t.Fatal("raw reflection was stored")
			}
		}
	}
}

func TestAPIValidationAndStaticPage(t *testing.T) {
	web := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<h1>晚安</h1>")}}
	client := &apiClient{t: t, handler: NewRouter(newMemoryStore(), &apiGenerator{}, web, false)}
	client.request(http.MethodPost, "/api/replies", `{"text":" "}`, http.StatusBadRequest)
	activationError := client.request(http.MethodPost, "/api/activate", `{"code":"WRONG"}`, http.StatusBadRequest)
	if activationError["error"] != "激活码无效或已被使用，请核对后再试。" {
		t.Fatalf("unclear activation error: %+v", activationError)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	client.handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "<h1>晚安</h1>" {
		t.Fatalf("static page failed: %d %s", response.Code, response.Body.String())
	}
}

func TestReplyRateLimit(t *testing.T) {
	client := &apiClient{t: t, handler: NewRouter(newMemoryStore(), &apiGenerator{}, fstest.MapFS{}, false)}
	for i := 0; i < 10; i++ {
		client.request(http.MethodPost, "/api/replies", `{"text":"今天很累"}`, http.StatusOK)
	}
	client.request(http.MethodPost, "/api/replies", `{"text":"今天很累"}`, http.StatusTooManyRequests)
}

var _ fs.FS = fstest.MapFS{}

func TestPublicBetaCompletesSevenDaysWithoutActivation(t *testing.T) {
	client := &apiClient{t: t, handler: NewRouterWithConfig(newMemoryStore(), &apiGenerator{}, fstest.MapFS{}, false, Config{PublicBeta: true, DailyReplyLimit: 20})}
	status := client.request(http.MethodPost, "/api/session", "", http.StatusOK)
	if status["publicBeta"] != true {
		t.Fatalf("public beta flag missing: %+v", status)
	}
	for day := 1; day <= 7; day++ {
		client.request(http.MethodPost, "/api/replies", fmt.Sprintf(`{"text":"公测第%d天"}`, day), http.StatusOK)
		client.request(http.MethodPost, "/api/entries", fmt.Sprintf(`{"day":%d,"mood":"平静","ritual":"揉掉"}`, day), http.StatusOK)
		if day == 2 {
			client.request(http.MethodPost, "/api/feedback", `{"day":2,"text":"揉掉文字的时候轻松了一点"}`, http.StatusOK)
			client.request(http.MethodPost, "/api/feedback", `{"day":2,"text":"更新后的反馈"}`, http.StatusOK)
		}
	}
	client.request(http.MethodPost, "/api/feedback", `{"day":7,"text":"七天回顾让我愿意继续"}`, http.StatusOK)
	client.request(http.MethodGet, "/api/review", "", http.StatusOK)
}

func TestPublicBetaDailyReplyLimit(t *testing.T) {
	client := &apiClient{t: t, handler: NewRouterWithConfig(newMemoryStore(), &apiGenerator{}, fstest.MapFS{}, false, Config{PublicBeta: true, DailyReplyLimit: 2})}
	client.request(http.MethodPost, "/api/replies", `{"text":"第一次"}`, http.StatusOK)
	client.request(http.MethodPost, "/api/replies", `{"text":"第二次"}`, http.StatusOK)
	client.request(http.MethodPost, "/api/replies", `{"text":"第三次"}`, http.StatusTooManyRequests)
}

func TestFeedbackValidation(t *testing.T) {
	client := &apiClient{t: t, handler: NewRouterWithConfig(newMemoryStore(), &apiGenerator{}, fstest.MapFS{}, false, Config{PublicBeta: true, DailyReplyLimit: 20})}
	client.request(http.MethodPost, "/api/feedback", `{"day":1,"text":"还没到反馈时间"}`, http.StatusBadRequest)
	client.request(http.MethodPost, "/api/feedback", `{"day":2,"text":" "}`, http.StatusBadRequest)
}
