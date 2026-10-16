package moviesearch

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	javDBMovieLinkPattern    = regexp.MustCompile(`(?i)href=["'](?:https?://[^/"']+)?(/v/[^"'?#]+)["']`)
	javDBItemStartPattern    = regexp.MustCompile(`(?is)<div\s+class=["']item(?:\s+odd)?\s*["'][^>]*>`)
	attributePattern         = regexp.MustCompile(`(?i)(data-rank|data-size|data-files|data-date)=["']([^"']*)["']`)
	javDBNamePattern         = regexp.MustCompile(`(?is)<span[^>]*class=["'][^"']*\bname\b[^"']*["'][^>]*>(.*?)</span>`)
	javDBInfoItemPattern     = regexp.MustCompile(`(?is)<div[^>]*class=["'][^"']*\b(?:panel-block|movie-panel-info-item)\b[^"']*["'][^>]*>(.*?)</div>`)
	javDBCoverPattern        = regexp.MustCompile(`(?is)<img[^>]+class=["'][^"']*\b(?:video-cover|cover)\b[^"']*["'][^>]+(?:src|data-src)=["']([^"']+)["']`)
	javDBCurrentTitlePattern = regexp.MustCompile(`(?is)<strong[^>]*class=["'][^"']*\bcurrent-title\b[^"']*["'][^>]*>(.*?)</strong>`)
	javDBTitlePattern        = regexp.MustCompile(`(?is)<h[1-4][^>]*class=["'][^"']*\btitle\b[^"']*["'][^>]*>(.*?)</h[1-4]>`)
)

type JavDBSource struct {
	client  *http.Client
	baseURL string
}

func NewJavDBSource(client *http.Client) *JavDBSource {
	return NewJavDBSourceWithDomain(client, DefaultJavDBDomain)
}

func NewJavDBSourceWithDomain(client *http.Client, domain string) *JavDBSource {
	if client == nil {
		client = defaultHTTPClient
	}
	return &JavDBSource{client: client, baseURL: baseURL(domain)}
}

func (s *JavDBSource) Name() string  { return "javdb" }
func (s *JavDBSource) Label() string { return "JavDB" }

func (s *JavDBSource) Search(ctx context.Context, keyword string) ([]Result, error) {
	searchURL := s.baseURL + "/search?q=" + url.QueryEscape(keyword) + "&f=all"
	searchHTML, err := s.get(ctx, searchURL)
	if err != nil {
		return nil, fmt.Errorf("JavDB 搜索失败: %w", err)
	}

	path := findJavDBMoviePath(searchHTML, keyword)
	if path == "" {
		return []Result{}, nil
	}

	detailHTML, err := s.get(ctx, s.baseURL+path)
	if err != nil {
		return nil, fmt.Errorf("JavDB 获取影片详情失败: %w", err)
	}
	metadata := parseJavDBMetadata(detailHTML, s.baseURL)
	if metadata.Number == "" {
		metadata.Number = strings.TrimSpace(keyword)
	}
	results := parseJavDBMagnets(detailHTML)
	for i := range results {
		applyMetadata(&results[i], metadata)
	}
	return results, nil
}

func (s *JavDBSource) get(ctx context.Context, target string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	applyBrowserHeaders(req, s.baseURL)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	return readResponse(resp)
}

func findJavDBMoviePath(body, keyword string) string {
	matches := javDBMovieLinkPattern.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return ""
	}
	keyword = strings.ToUpper(strings.TrimSpace(keyword))
	for _, match := range matches {
		if len(match) > 1 && strings.Contains(strings.ToUpper(match[1]), keyword) {
			return match[1]
		}
	}
	return matches[0][1]
}

func parseJavDBMagnets(body string) []Result {
	items := splitJavDBItems(body)
	type rankedResult struct {
		rank   int
		result Result
	}
	ranked := make([]rankedResult, 0, len(items))

	for _, item := range items {
		magnetMatch := magnetPattern.FindStringSubmatch(item)
		if len(magnetMatch) < 2 {
			continue
		}
		attributes := make(map[string]string)
		for _, match := range attributePattern.FindAllStringSubmatch(item, -1) {
			attributes[strings.ToLower(match[1])] = match[2]
		}

		magnet := normalizeMagnet(magnetMatch[1])
		name := magnetDisplayName(magnet)
		if nameMatch := javDBNamePattern.FindStringSubmatch(item); len(nameMatch) > 1 {
			if parsedName := decodeHTMLText(nameMatch[1]); parsedName != "" {
				name = parsedName
			}
		}

		files, _ := strconv.Atoi(attributes["data-files"])
		rank, _ := strconv.Atoi(attributes["data-rank"])
		result := Result{
			Hash:   extractHash(magnet),
			Name:   name,
			Size:   formatSizeMB(attributes["data-size"]),
			Date:   formatCompactDate(attributes["data-date"]),
			Files:  files,
			Tags:   parseTags(item),
			Magnet: magnet,
		}
		ranked = append(ranked, rankedResult{rank: rank, result: result})
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].rank < ranked[j].rank
	})
	results := make([]Result, len(ranked))
	for i := range ranked {
		results[i] = ranked[i].result
	}
	return results
}

func parseJavDBMetadata(body, baseURL string) Result {
	metadata := Result{}
	if match := javDBCoverPattern.FindStringSubmatch(body); len(match) > 1 {
		metadata.CoverURL = resolveImageURL(baseURL, match[1])
	}
	if match := javDBCurrentTitlePattern.FindStringSubmatch(body); len(match) > 1 {
		metadata.Title = normalizeHTMLText(match[1])
	} else if match := javDBTitlePattern.FindStringSubmatch(body); len(match) > 1 {
		metadata.Title = normalizeHTMLText(match[1])
	}
	metadata.Number = extractMovieNumber(metadata.Title)

	blocks := javDBInfoItemPattern.FindAllStringSubmatch(body, -1)
	for _, match := range blocks {
		if len(match) < 2 {
			continue
		}
		block := match[1]
		if value, ok := extractLabeledBlockValue(block, "發行日期", "发行日期", "日期"); ok {
			metadata.ReleaseDate = firstDate(value)
		}
		if value, ok := extractLabeledBlockValue(block, "時長", "时长", "長度", "长度"); ok {
			metadata.Runtime = parseRuntime(value)
		}
		if value, ok := extractLabeledBlockValue(block, "片商", "製作商", "制作商"); ok {
			metadata.Maker = value
		}
		if value, ok := extractLabeledBlockValue(block, "發行商", "发行商", "發行", "发行"); ok {
			metadata.Label = value
		}
		if value, ok := extractLabeledBlockValue(block, "系列"); ok {
			metadata.Series = value
		}
		if value, ok := extractLabeledBlockValue(block, "導演", "导演"); ok {
			metadata.Director = value
		}
		if actors := extractLabeledBlockAnchors(block, "演員", "演员"); len(actors) > 0 {
			metadata.Actors = actors
		}
		if categories := extractLabeledBlockAnchors(block, "類別", "类别", "類型", "类型"); len(categories) > 0 {
			metadata.Categories = categories
		}
	}
	return metadata
}

func splitJavDBItems(body string) []string {
	locations := javDBItemStartPattern.FindAllStringIndex(body, -1)
	items := make([]string, 0, len(locations))
	for i, location := range locations {
		end := len(body)
		if i+1 < len(locations) {
			end = locations[i+1][0]
		}
		items = append(items, body[location[0]:end])
	}
	return items
}

func parseTags(body string) []string {
	matches := tagPattern.FindAllStringSubmatch(body, -1)
	tags := make([]string, 0, len(matches))
	for _, match := range matches {
		if tag := decodeHTMLText(match[1]); tag != "" {
			if tag == "是" || tag == "否" {
				continue
			}
			tags = append(tags, tag)
		}
	}
	return tags
}
