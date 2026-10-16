package moviesearch

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

const DefaultSource = "javbus"

// Result 是不同搜索源统一返回的磁链结果。
type Result struct {
	Source      string   `json:"source"`
	Hash        string   `json:"hash"`
	Name        string   `json:"name"`
	Size        string   `json:"size"`
	Date        string   `json:"date"`
	Files       int      `json:"files,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Magnet      string   `json:"magnet"`
	CoverURL    string   `json:"cover_url,omitempty"`
	Number      string   `json:"number,omitempty"`
	Categories  []string `json:"categories,omitempty"`
	Title       string   `json:"title,omitempty"`
	Maker       string   `json:"maker,omitempty"`
	ReleaseDate string   `json:"release_date,omitempty"`
	Runtime     int      `json:"runtime,omitempty"`
	Series      string   `json:"series,omitempty"`
	Label       string   `json:"label,omitempty"`
	Director    string   `json:"director,omitempty"`
	Actors      []string `json:"actors,omitempty"`
}

// Source 描述一个可注册的影片搜索源。
type Source interface {
	Name() string
	Label() string
	Search(ctx context.Context, keyword string) ([]Result, error)
}

type SourceInfo struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}

// Service 根据来源名称分派搜索请求。
type Service struct {
	sources map[string]Source
}

func NewService(sources ...Source) *Service {
	service := &Service{sources: make(map[string]Source, len(sources))}
	for _, source := range sources {
		if source == nil {
			continue
		}
		service.sources[strings.ToLower(source.Name())] = source
	}
	return service
}

func NewDefaultService() *Service {
	return NewConfiguredService(DefaultJavBusDomain, DefaultJavDBDomain)
}

func NewConfiguredService(javBusDomain, javDBDomain string) *Service {
	return NewService(
		NewJavBusSourceWithDomain(nil, javBusDomain),
		NewJavDBSourceWithDomain(nil, javDBDomain),
	)
}

func (s *Service) Search(ctx context.Context, sourceName, keyword string) ([]Result, error) {
	sourceName = strings.ToLower(strings.TrimSpace(sourceName))
	source, ok := s.sources[sourceName]
	if !ok {
		return nil, fmt.Errorf("不支持的搜索源: %s", sourceName)
	}

	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("搜索关键词不能为空")
	}

	results, err := source.Search(ctx, keyword)
	if err != nil {
		return nil, err
	}
	for i := range results {
		results[i].Source = source.Name()
	}
	return results, nil
}

func (s *Service) HasSource(name string) bool {
	_, ok := s.sources[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

func (s *Service) Sources() []SourceInfo {
	result := make([]SourceInfo, 0, len(s.sources))
	for _, source := range s.sources {
		result = append(result, SourceInfo{Name: source.Name(), Label: source.Label()})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}
