package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"unwind/internal/plan"
)

type sessionStore struct {
	ids []string
}

func (s *sessionStore) EnsureDevice(_ context.Context, id string) error {
	s.ids = append(s.ids, id)
	return nil
}
func (s *sessionStore) Progress(context.Context, string) (plan.Progress, error) {
	return plan.Progress{}, nil
}
func (s *sessionStore) Complete(context.Context, string, int, string, string) (plan.Entry, error) {
	return plan.Entry{}, nil
}
func (s *sessionStore) Activate(context.Context, string, string) error { return nil }

func TestSessionIssueAndRestore(t *testing.T) {
	store := &sessionStore{}
	manager := NewSessionManager(store, false)
	first := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	deviceID, err := manager.Ensure(response, first)
	if err != nil {
		t.Fatal(err)
	}
	if deviceID == "" || len(store.ids) != 1 || store.ids[0] == "" {
		t.Fatalf("session was not persisted: id=%q ids=%v", deviceID, store.ids)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected session cookie: %+v", cookies)
	}

	second := httptest.NewRequest(http.MethodGet, "/", nil)
	second.AddCookie(cookies[0])
	restored, err := manager.Ensure(httptest.NewRecorder(), second)
	if err != nil {
		t.Fatal(err)
	}
	if restored != deviceID {
		t.Fatalf("restored %q, want %q", restored, deviceID)
	}
}

func TestSessionReplacesMalformedCookie(t *testing.T) {
	manager := NewSessionManager(&sessionStore{}, false)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "short"})
	response := httptest.NewRecorder()
	if _, err := manager.Ensure(response, request); err != nil {
		t.Fatal(err)
	}
	if len(response.Result().Cookies()) != 1 {
		t.Fatal("malformed cookie should be replaced")
	}
}
