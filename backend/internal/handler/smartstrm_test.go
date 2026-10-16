package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSmartStrmConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			if r.Method != http.MethodPost {
				t.Errorf("login method = %s, want POST", r.Method)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse login form: %v", err)
			}
			if r.Form.Get("username") != "tester" || r.Form.Get("password") != "secret" {
				t.Errorf("unexpected login form: %#v", r.Form)
			}
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "valid", Path: "/"})
			http.Redirect(w, r, "/", http.StatusFound)
		case "/api/version":
			cookie, err := r.Cookie("session")
			if err != nil || cookie.Value != "valid" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(smartStrmVersion{
				Version:       "v0.5.0",
				RemoteVersion: "v0.5.0",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	version, err := testSmartStrmConnection(
		context.Background(),
		server.URL,
		"tester",
		"secret",
	)
	if err != nil {
		t.Fatalf("test connection: %v", err)
	}
	if version.Version != "v0.5.0" {
		t.Fatalf("version = %q, want v0.5.0", version.Version)
	}
}

func TestSmartStrmConnectionRejectsLoginWithoutRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := testSmartStrmConnection(
		context.Background(),
		server.URL,
		"tester",
		"wrong",
	)
	if err == nil {
		t.Fatal("expected login error")
	}
}

func TestFetchSmartStrmTasks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "valid", Path: "/"})
			http.Redirect(w, r, "/", http.StatusFound)
		case "/api/tasks":
			cookie, err := r.Cookie("session")
			if err != nil || cookie.Value != "valid" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{
				"电影": {
					"name": "电影",
					"config": {"name": "电影", "storage": "光鸭"},
					"next_run": null,
					"running": false
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tasks, err := fetchSmartStrmTasks(
		context.Background(),
		server.URL,
		"tester",
		"secret",
	)
	if err != nil {
		t.Fatalf("fetch tasks: %v", err)
	}
	task, ok := tasks["电影"]
	if !ok || task.Name != "电影" || task.Running {
		t.Fatalf("unexpected tasks: %#v", tasks)
	}
}
