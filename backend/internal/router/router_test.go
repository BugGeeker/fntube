package router

import (
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
)

func TestStripGatewayPrefix(t *testing.T) {
	tests := map[string]string{
		"/app/FnTube":                         "/",
		"/app/FnTube/":                        "/",
		"/app/FnTube/index.html":              "/index.html",
		"/app/FnTube/assets/index-example.js": "/assets/index-example.js",
		"/assets/index-example.js":            "/assets/index-example.js",
	}

	for requestPath, expected := range tests {
		t.Run(requestPath, func(t *testing.T) {
			ctx := app.NewContext(0)
			ctx.Request.SetRequestURI(requestPath)
			if actual := string(stripGatewayPrefix(ctx)); actual != expected {
				t.Fatalf("stripGatewayPrefix(%q) = %q, want %q", requestPath, actual, expected)
			}
		})
	}
}
