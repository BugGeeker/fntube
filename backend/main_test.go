package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

func TestUnifiedGatewaySocket(t *testing.T) {
	appDest := t.TempDir()
	t.Setenv("TRIM_APPDEST", appDest)
	t.Setenv("FNTUBE_UNIFIED_GATEWAY", "1")

	h, cleanup, err := createServer()
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	t.Cleanup(cleanup)

	h.GET("/api/health", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- h.Run()
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = h.Shutdown(ctx)
	})

	socketPath := filepath.Join(appDest, "app.sock")
	for _, requestPath := range []string{
		"/app/FnTube/api/health",
		"/api/health",
	} {
		t.Run(requestPath, func(t *testing.T) {
			var conn net.Conn
			for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
				conn, err = net.DialTimeout("unix", socketPath, 100*time.Millisecond)
				if err == nil {
					break
				}
				select {
				case runErr := <-errCh:
					t.Fatalf("server stopped before accepting requests: %v", runErr)
				default:
				}
				time.Sleep(20 * time.Millisecond)
			}
			if err != nil {
				t.Fatalf("connect to gateway socket: %v", err)
			}
			defer conn.Close()

			if _, err := fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n", requestPath); err != nil {
				t.Fatalf("write request: %v", err)
			}
			response, err := http.ReadResponse(bufio.NewReader(conn), nil)
			if err != nil {
				t.Fatalf("read response: %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != http.StatusOK {
				t.Fatalf("unexpected status: %s", response.Status)
			}
			var body map[string]string
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body["status"] != "ok" {
				t.Fatalf("unexpected response body: %#v", body)
			}
		})
	}
}
