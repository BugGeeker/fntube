package moviesearch

import (
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	magnetPattern      = regexp.MustCompile(`(?i)href=["'](magnet:\?[^"']+)["']`)
	hashPattern        = regexp.MustCompile(`(?i)btih:([a-z0-9]+)`)
	datePattern        = regexp.MustCompile(`\b(\d{4}-\d{2}-\d{2})\b`)
	sizePattern        = regexp.MustCompile(`(?i)\b(\d+(?:\.\d+)?\s*(?:KB|MB|GB|TB))\b`)
	tagPattern         = regexp.MustCompile(`(?is)<span[^>]*class=["'][^"']*\btag\b[^"']*["'][^>]*>(.*?)</span>`)
	htmlTagPattern     = regexp.MustCompile(`(?s)<[^>]+>`)
	anchorPattern      = regexp.MustCompile(`(?is)<a[^>]*>(.*?)</a>`)
	movieNumberPattern = regexp.MustCompile(`(?i)\b[A-Z]{2,10}[-_ ]?\d{2,5}\b`)
)

func decodeHTMLText(value string) string {
	value = htmlTagPattern.ReplaceAllString(value, "")
	return strings.TrimSpace(html.UnescapeString(value))
}

func normalizeMagnet(value string) string {
	return html.UnescapeString(strings.TrimSpace(value))
}

func extractHash(magnet string) string {
	match := hashPattern.FindStringSubmatch(magnet)
	if len(match) < 2 {
		return ""
	}
	return strings.ToUpper(match[1])
}

func magnetDisplayName(magnet string) string {
	parsed, err := url.Parse(magnet)
	if err != nil {
		return magnet
	}
	if name := parsed.Query().Get("dn"); name != "" {
		return name
	}
	return magnet
}

func formatSizeMB(raw string) string {
	size, err := strconv.Atoi(raw)
	if err != nil {
		return ""
	}
	if size >= 1024 {
		return strconv.FormatFloat(float64(size)/1024, 'f', 2, 64) + "GB"
	}
	return strconv.Itoa(size) + "MB"
}

func formatCompactDate(raw string) string {
	if len(raw) != 8 {
		return raw
	}
	return raw[:4] + "-" + raw[4:6] + "-" + raw[6:]
}

func normalizeHTMLText(value string) string {
	return strings.Join(strings.Fields(decodeHTMLText(value)), " ")
}

func extractMovieNumber(value string) string {
	match := movieNumberPattern.FindString(value)
	if match == "" {
		return ""
	}
	return strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(match), "_", "-"), " ", "-")
}

func extractLabeledBlockValue(block string, labels ...string) (string, bool) {
	plain := normalizeHTMLText(block)
	for _, label := range labels {
		index := strings.Index(plain, label)
		if index < 0 {
			continue
		}
		value := strings.TrimSpace(plain[index+len(label):])
		value = strings.TrimLeft(value, " :：")
		if value != "" {
			return value, true
		}
	}
	return "", false
}

func extractLabeledBlockAnchors(block string, labels ...string) []string {
	plain := normalizeHTMLText(block)
	hasLabel := false
	for _, label := range labels {
		if strings.Contains(plain, label) {
			hasLabel = true
			break
		}
	}
	if !hasLabel {
		return nil
	}

	anchors := make([]string, 0)
	for _, match := range anchorPattern.FindAllStringSubmatch(block, -1) {
		if len(match) < 2 {
			continue
		}
		if value := normalizeHTMLText(match[1]); value != "" {
			anchors = append(anchors, value)
		}
	}
	return uniqueStrings(anchors)
}

func parseRuntime(value string) int {
	match := regexp.MustCompile(`(?i)(\d+)\s*(?:分钟|分|min)`).FindStringSubmatch(value)
	if len(match) < 2 {
		return 0
	}
	runtime, _ := strconv.Atoi(match[1])
	return runtime
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func resolveImageURL(baseURL, imageURL string) string {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return ""
	}
	parsed, err := url.Parse(imageURL)
	if err != nil {
		return imageURL
	}
	if parsed.IsAbs() {
		return parsed.String()
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return imageURL
	}
	return base.ResolveReference(parsed).String()
}
