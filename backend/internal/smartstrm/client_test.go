package smartstrm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "valid", Path: "/"})
			http.Redirect(w, r, "/", http.StatusFound)
		case "/api/tasks/日本/run":
			if r.Method != http.MethodPost {
				t.Errorf("method = %s, want POST", r.Method)
			}
			cookie, err := r.Cookie("session")
			if err != nil || cookie.Value != "valid" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			if payload["storage_path"] != "/日本/有码/T28/T28-205" {
				t.Errorf("storage_path = %q", payload["storage_path"])
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	err := RunTask(
		context.Background(),
		server.URL,
		"tester",
		"secret",
		"日本",
		"/日本/有码/T28/T28-205",
	)
	if err != nil {
		t.Fatalf("run task: %v", err)
	}
}
