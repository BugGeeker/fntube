package moviesearch

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var (
	javBusVideoLinkPattern = regexp.MustCompile(`(?i)href=["'](?:https?://[^/"']+)?/([a-z0-9-]+)["']`)
	javBusGIDPattern       = regexp.MustCompile(`(?i)var\s+gid\s*=\s*(\d+)`)
	javBusUCPattern        = regexp.MustCompile(`(?i)var\s+uc\s*=\s*(\d+)`)
	javBusRowPattern       = regexp.MustCompile(`(?is)<tr[^>]*>(.*?)</tr>`)
	javBusNamePattern      = regexp.MustCompile(`(?is)<a[^>]*href=["']magnet:\?[^"']+["'][^>]*>(.*?)</a>`)
	javBusParagraphPattern = regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`)
	javBusCoverPattern     = regexp.MustCompile(`(?is)<(?:div|a)[^>]*class=["'][^"']*\b(?:photo-frame|bigImage)\b[^"']*["'][^>]*>.*?<img[^>]+src=["']([^"']+)["']`)
	javBusTitlePattern     = regexp.MustCompile(`(?is)<h[1-3][^>]*>(.*?)</h[1-3]>`)
	javBusActorPattern     = regexp.MustCompile(`(?is)<div[^>]*class=["'][^"']*\bstar-name\b[^"']*["'][^>]*>\s*<a[^>]*>(.*?)</a>\s*</div>`)
)

type JavBusSource struct {
	client  *http.Client
	baseURL string
}

func NewJavBusSource(client *http.Client) *JavBusSource {
	return NewJavBusSourceWithDomain(client, DefaultJavBusDomain)
}

func NewJavBusSourceWithDomain(client *http.Client, domain string) *JavBusSource {
	if client == nil {
		client = defaultHTTPClient
	}
	return &JavBusSource{client: client, baseURL: baseURL(domain)}
}

func (s *JavBusSource) Name() string  { return "javbus" }
func (s *JavBusSource) Label() string { return "JavBus" }

func (s *JavBusSource) Search(ctx context.Context, keyword string) ([]Result, error) {
	searchHTML, err := s.get(ctx, s.baseURL+"/search/"+url.PathEscape(keyword), s.baseURL, false)
	if err != nil {
		return nil, fmt.Errorf("JavBus 搜索失败: %w", err)
	}

	videoCode := findJavBusVideoCode(searchHTML, keyword)
	if videoCode == "" {
		return []Result{}, nil
	}

	detailURL := s.baseURL + "/" + videoCode
	detailHTML, err := s.get(ctx, detailURL, s.baseURL, false)
	if err != nil {
		return nil, fmt.Errorf("JavBus 获取影片详情失败: %w", err)
	}
	metadata := parseJavBusMetadata(detailHTML, s.baseURL)
	if metadata.Number == "" {
		metadata.Number = strings.TrimSpace(keyword)
	}

	gid := firstMatch(javBusGIDPattern, detailHTML)
	uc := firstMatch(javBusUCPattern, detailHTML)
	if gid == "" || uc == "" {
		return nil, fmt.Errorf("JavBus 影片页面缺少磁链查询参数")
	}

	params := url.Values{
		"gid":   {gid},
		"uc":    {uc},
		"lang":  {"zh"},
		"floor": {"500"},
	}
	magnetHTML, err := s.get(ctx, s.baseURL+"/ajax/uncledatoolsbyajax.php?"+params.Encode(), detailURL, true)
	if err != nil {
		return nil, fmt.Errorf("JavBus 获取磁链失败: %w", err)
	}
	results := parseJavBusMagnets(magnetHTML)
	for i := range results {
		applyMetadata(&results[i], metadata)
	}
	return results, nil
}

func (s *JavBusSource) get(ctx context.Context, target, referer string, ajax bool) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	applyBrowserHeaders(req, referer)
	req.Header.Set("Cookie", "over18=1; existmag=mag")
	if ajax {
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	return readResponse(resp)
}

func findJavBusVideoCode(body, keyword string) string {
	keyword = strings.ToUpper(strings.TrimSpace(keyword))
	matches := javBusVideoLinkPattern.FindAllStringSubmatch(body, -1)
	for _, match := range matches {
		if len(match) > 1 && strings.Contains(strings.ToUpper(match[1]), keyword) {
			return match[1]
		}
	}
	return ""
}

func parseJavBusMagnets(body string) []Result {
	rows := javBusRowPattern.FindAllStringSubmatch(body, -1)
	results := make([]Result, 0, len(rows))
	for _, row := range rows {
		magnetMatch := magnetPattern.FindStringSubmatch(row[1])
		if len(magnetMatch) < 2 {
			continue
		}
		magnet := normalizeMagnet(magnetMatch[1])
		name := magnetDisplayName(magnet)
		if nameMatch := javBusNamePattern.FindStringSubmatch(row[1]); len(nameMatch) > 1 {
			if parsedName := decodeHTMLText(nameMatch[1]); parsedName != "" {
				name = parsedName
			}
		}

		result := Result{Hash: extractHash(magnet), Name: name, Magnet: magnet}
		if sizeMatch := sizePattern.FindStringSubmatch(row[1]); len(sizeMatch) > 1 {
			result.Size = strings.ReplaceAll(strings.ToUpper(sizeMatch[1]), " ", "")
		}
		if dateMatch := datePattern.FindStringSubmatch(row[1]); len(dateMatch) > 1 {
			result.Date = dateMatch[1]
		}
		results = append(results, result)
	}
	return results
}

func parseJavBusMetadata(body, baseURL string) Result {
	metadata := Result{}
	if match := javBusCoverPattern.FindStringSubmatch(body); len(match) > 1 {
		metadata.CoverURL = resolveImageURL(baseURL, match[1])
	}
	if match := javBusTitlePattern.FindStringSubmatch(body); len(match) > 1 {
		metadata.Title = normalizeHTMLText(match[1])
	}
	metadata.Number = extractMovieNumber(metadata.Title)

	for _, match := range javBusParagraphPattern.FindAllStringSubmatch(body, -1) {
		if len(match) < 2 {
			continue
		}
		block := match[1]
		if value, ok := extractLabeledBlockValue(block, "發行日期", "发行日期", "日期"); ok {
			metadata.ReleaseDate = firstDate(value)
		}
		if value, ok := extractLabeledBlockValue(block, "長度", "长度", "時長", "时长"); ok {
			metadata.Runtime = parseRuntime(value)
		}
		if value, ok := extractLabeledBlockValue(block, "製作商", "制作商", "片商"); ok {
			metadata.Maker = value
		}
		if value, ok := extractLabeledBlockValue(block, "發行商", "发行商", "片商"); ok {
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
	for _, match := range javBusActorPattern.FindAllStringSubmatch(body, -1) {
		if len(match) < 2 {
			continue
		}
		if actor := normalizeHTMLText(match[1]); actor != "" {
			metadata.Actors = append(metadata.Actors, actor)
		}
	}
	metadata.Actors = uniqueStrings(metadata.Actors)
	return metadata
}

func firstDate(value string) string {
	match := datePattern.FindString(value)
	return match
}

func applyMetadata(result *Result, metadata Result) {
	result.CoverURL = metadata.CoverURL
	result.Number = metadata.Number
	result.Categories = metadata.Categories
	result.Title = metadata.Title
	result.Maker = metadata.Maker
	result.ReleaseDate = metadata.ReleaseDate
	result.Runtime = metadata.Runtime
	result.Series = metadata.Series
	result.Label = metadata.Label
	result.Director = metadata.Director
	result.Actors = metadata.Actors
}

func firstMatch(pattern *regexp.Regexp, value string) string {
	match := pattern.FindStringSubmatch(value)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}
