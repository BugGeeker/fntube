package moviesearch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubSource struct{}

func (stubSource) Name() string  { return "stub" }
func (stubSource) Label() string { return "Stub" }
func (stubSource) Search(_ context.Context, _ string) ([]Result, error) {
	return []Result{{Name: "result"}}, nil
}

func TestServiceDispatchesBySourceName(t *testing.T) {
	service := NewService(stubSource{})

	results, err := service.Search(context.Background(), "STUB", "ABC-123")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 1 || results[0].Source != "stub" {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestServiceRejectsUnknownSource(t *testing.T) {
	service := NewService(stubSource{})

	if _, err := service.Search(context.Background(), "missing", "ABC-123"); err == nil {
		t.Fatal("expected unsupported source error")
	}
}

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "domain", value: "www.javdb.com", want: "www.javdb.com"},
		{name: "trim and lowercase", value: "  WWW.JAVBUS.COM  ", want: "www.javbus.com"},
		{name: "domain with port", value: "example.com:8443", want: "example.com:8443"},
		{name: "protocol", value: "https://www.javdb.com", wantErr: true},
		{name: "path", value: "www.javdb.com/search", wantErr: true},
		{name: "query", value: "www.javdb.com?q=ABC-123", wantErr: true},
		{name: "fragment", value: "www.javdb.com#results", wantErr: true},
		{name: "authentication", value: "user@example.com", wantErr: true},
		{name: "empty", value: "  ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeDomain(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NormalizeDomain(%q) expected error, got %q", tt.value, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeDomain(%q) returned error: %v", tt.value, err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeDomain(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestConfiguredSourcesUseCustomDomains(t *testing.T) {
	javBus := NewJavBusSourceWithDomain(nil, "bus.example.com")
	if javBus.baseURL != "https://bus.example.com" {
		t.Fatalf("unexpected JavBus base URL: %s", javBus.baseURL)
	}

	javDB := NewJavDBSourceWithDomain(nil, "db.example.com:8443")
	if javDB.baseURL != "https://db.example.com:8443" {
		t.Fatalf("unexpected JavDB base URL: %s", javDB.baseURL)
	}
}

func TestParseJavBusMagnets(t *testing.T) {
	body := `
<table>
  <tr>
    <td><a href="magnet:?xt=urn:btih:abcdef123456&amp;dn=ABC-123">ABC-123 高清</a></td>
    <td>1.25 GB</td>
    <td>2026-10-14</td>
  </tr>
</table>`

	results := parseJavBusMagnets(body)
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	got := results[0]
	if got.Hash != "ABCDEF123456" || got.Name != "ABC-123 高清" || got.Size != "1.25GB" || got.Date != "2026-10-14" {
		t.Fatalf("unexpected result: %#v", got)
	}
	if got.Magnet != "magnet:?xt=urn:btih:abcdef123456&dn=ABC-123" {
		t.Fatalf("magnet was not decoded: %s", got.Magnet)
	}
}

func TestParseJavBusMetadata(t *testing.T) {
	body := `
<a class="bigImage" href="/img/abc.jpg"><img src="/img/abc-thumb.jpg"></a>
<h3>ABC-123 影片标题</h3>
<p><b>发行日期:</b> 2026-10-14</p>
<p><b>长度:</b> 120 分钟</p>
<p><b>制作商:</b> <a href="/maker/test">测试片商</a></p>
<p><b>发行商:</b> <a href="/label/test">测试发行商</a></p>
<p><b>系列:</b> <a href="/series/test">测试系列</a></p>
<p><b>导演:</b> <a href="/director/test">测试导演</a></p>
<p><b>演员:</b></p>
<div class="star-name"><a href="/star/1">演员甲</a></div>
<div class="star-name"><a href="/star/2">演员乙</a></div>`

	metadata := parseJavBusMetadata(body, "https://www.javbus.com")
	if metadata.CoverURL != "https://www.javbus.com/img/abc-thumb.jpg" ||
		metadata.Title != "ABC-123 影片标题" ||
		metadata.Maker != "测试片商" ||
		metadata.Label != "测试发行商" ||
		metadata.ReleaseDate != "2026-10-14" ||
		metadata.Runtime != 120 ||
		metadata.Series != "测试系列" ||
		metadata.Director != "测试导演" {
		t.Fatalf("unexpected JavBus metadata: %#v", metadata)
	}
	if len(metadata.Actors) != 2 || metadata.Actors[0] != "演员甲" || metadata.Actors[1] != "演员乙" {
		t.Fatalf("unexpected JavBus actors: %#v", metadata.Actors)
	}
}

func TestParseJavDBMagnets(t *testing.T) {
	body := `
<div class="item odd" data-rank="1" data-size="2048" data-files="3" data-date="20261014">
  <a href="magnet:?xt=urn:btih:1234abcd&amp;dn=ABC-123">
    <span class="name">ABC-123 完整版</span>
  </a>
  <span class="tag is-warning">字幕</span>
  <span class="tag is-light">是</span>
  <span class="tag is-light">否</span>
</div>
<div class="item " data-rank="0" data-size="700" data-files="1" data-date="20261013">
  <a href="magnet:?xt=urn:btih:ffff0000&amp;dn=ABC-123-small">
    <span class="name">ABC-123 精简版</span>
  </a>
</div>`

	results := parseJavDBMagnets(body)
	if len(results) != 2 {
		t.Fatalf("expected two results, got %d", len(results))
	}
	if results[0].Name != "ABC-123 精简版" || results[0].Size != "700MB" || results[0].Date != "2026-10-13" {
		t.Fatalf("unexpected first result: %#v", results[0])
	}
	if results[1].Name != "ABC-123 完整版" || results[1].Size != "2.00GB" || results[1].Files != 3 {
		t.Fatalf("unexpected second result: %#v", results[1])
	}
	if len(results[1].Tags) != 1 || results[1].Tags[0] != "字幕" {
		t.Fatalf("unexpected tags: %#v", results[1].Tags)
	}
}

func TestParseJavDBMetadata(t *testing.T) {
	body := `
<div class="movie-panel">
  <img class="video-cover" src="/covers/abc.jpg">
  <h2 class="title is-4"><strong class="current-title">ABC-123 影片标题</strong></h2>
  <div class="panel-block"><strong>日期:</strong><span>2026-10-14</span></div>
  <div class="panel-block"><strong>時長:</strong><span>98 分鍾</span></div>
  <div class="panel-block"><strong>片商:</strong><a href="/maker/test">测试片商</a></div>
  <div class="panel-block"><strong>發行:</strong><a href="/label/test">测试发行商</a></div>
  <div class="panel-block"><strong>系列:</strong><a href="/series/test">测试系列</a></div>
  <div class="panel-block"><strong>導演:</strong><a href="/director/test">测试导演</a></div>
  <div class="panel-block"><strong>演員:</strong><a href="/star/1">演员甲</a><a href="/star/2">演员乙</a></div>
</div>`

	metadata := parseJavDBMetadata(body, "https://www.javdb.com")
	if metadata.CoverURL != "https://www.javdb.com/covers/abc.jpg" ||
		metadata.Title != "ABC-123 影片标题" ||
		metadata.Maker != "测试片商" ||
		metadata.Label != "测试发行商" ||
		metadata.ReleaseDate != "2026-10-14" ||
		metadata.Runtime != 98 ||
		metadata.Series != "测试系列" ||
		metadata.Director != "测试导演" {
		t.Fatalf("unexpected JavDB metadata: %#v", metadata)
	}
	if len(metadata.Actors) != 2 || metadata.Actors[0] != "演员甲" || metadata.Actors[1] != "演员乙" {
		t.Fatalf("unexpected JavDB actors: %#v", metadata.Actors)
	}
}

func TestJavBusSourceSearchFlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/ABC-123":
			_, _ = w.Write([]byte(`<a href="/ABC-123">ABC-123</a>`))
		case "/ABC-123":
			_, _ = w.Write([]byte(`var gid = 42; var uc = 1;`))
		case "/ajax/uncledatoolsbyajax.php":
			if r.URL.Query().Get("gid") != "42" || r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
				t.Fatalf("unexpected JavBus AJAX request: %s headers=%v", r.URL.String(), r.Header)
			}
			_, _ = w.Write([]byte(`<tr><td><a href="magnet:?xt=urn:btih:aabbccdd&amp;dn=ABC-123">ABC-123</a></td><td>1 GB</td><td>2026-10-14</td></tr>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	source := &JavBusSource{client: server.Client(), baseURL: server.URL}
	results, err := source.Search(context.Background(), "ABC-123")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 1 || results[0].Hash != "AABBCCDD" {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestJavDBSourceSearchFlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search":
			if r.URL.Query().Get("q") != "ABC-123" {
				t.Fatalf("unexpected search query: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`<a href="/v/ABC-123">ABC-123</a>`))
		case "/v/ABC-123":
			_, _ = w.Write([]byte(`
<div class="item odd" data-rank="0" data-size="1024" data-files="1" data-date="20261014">
  <a href="magnet:?xt=urn:btih:11223344&amp;dn=ABC-123"><span class="name">ABC-123</span></a>
</div>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	source := &JavDBSource{client: server.Client(), baseURL: server.URL}
	results, err := source.Search(context.Background(), "ABC-123")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 1 || results[0].Size != "1.00GB" {
		t.Fatalf("unexpected results: %#v", results)
	}
}
