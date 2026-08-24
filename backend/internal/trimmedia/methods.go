package trimmedia

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

// MediaDbList 媒体库列表(普通用户)
func (c *Client) MediaDbList() ([]MediaDb, error) {
	res, err := c.request("/mediadb/list", "GET", map[string]string{"page_size": "9999"}, nil, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("mediadb list failed: %v", err)
	}
	return c.parseMediaDbList(res.Data, "title"), nil
}

// MdbList 媒体库列表(管理员)
func (c *Client) MdbList() ([]MediaDb, error) {
	res, err := c.request("/mdb/list", "GET", map[string]string{"page_size": "9999"}, nil, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("mdb list failed: %v", err)
	}
	return c.parseMediaDbList(res.Data, "name"), nil
}

// parseMediaDbList 解析媒体库列表
// nameField: 普通用户用 "title"，管理员用 "name"
func (c *Client) parseMediaDbList(data interface{}, nameField string) []MediaDb {
	if data == nil {
		return []MediaDb{}
	}
	// data 可能是 map 或直接是数组
	var listRaw interface{}
	if m, ok := data.(map[string]interface{}); ok {
		if l, ok2 := m["list"]; ok2 {
			listRaw = l
		} else {
			listRaw = data
		}
	} else {
		listRaw = data
	}
	arr, err := asInterfaceList(listRaw)
	if err != nil {
		return []MediaDb{}
	}
	items := make([]MediaDb, 0, len(arr))
	for _, info := range arr {
		mdb := MediaDb{}
		mdb.GUID = getString(info, "guid")
		mdb.Category = ParseCategory(getString(info, "category"))
		mdb.Name = getString(info, nameField)
		if posters := getStringList(info, "posters"); len(posters) > 0 {
			for _, p := range posters {
				mdb.Posters = append(mdb.Posters, c.buildImgAPIURL(p))
			}
		}
		mdb.DirList = getStringList(info, "dir_list")
		items = append(items, mdb)
	}
	return items
}

// MdbScanAll 扫描所有媒体库
func (c *Client) MdbScanAll() (bool, error) {
	res, err := c.request("/mdb/scanall", "POST", nil, map[string]interface{}{}, "", false)
	if err != nil || !res.Success() {
		return false, fmt.Errorf("mdb scanall failed: %v", err)
	}
	return res.Data != nil, nil
}

// MdbScan 扫描指定媒体库
func (c *Client) MdbScan(guid string) (bool, error) {
	res, err := c.request(fmt.Sprintf("/mdb/scan/%s", guid), "POST", nil, map[string]interface{}{}, "", false)
	if err != nil || !res.Success() {
		return false, fmt.Errorf("mdb scan failed: %v", err)
	}
	return res.Data != nil, nil
}

// MediaDbSum 媒体数量统计
func (c *Client) MediaDbSum() (*MediaDbSummary, error) {
	res, err := c.request("/mediadb/sum", "GET", nil, nil, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("mediadb sum failed: %v", err)
	}
	if res.Data == nil {
		return nil, nil
	}
	summary := &MediaDbSummary{}
	summary.Movie = getInt(res.DataMap(), "movie")
	summary.TV = getInt(res.DataMap(), "tv")
	summary.Video = getInt(res.DataMap(), "video")
	summary.Favorite = getInt(res.DataMap(), "favorite")
	summary.Total = getInt(res.DataMap(), "total")
	return summary, nil
}

// ItemList 媒体列表
// types: 需要查询的类型列表，为空时使用默认 [Movie, TV, Directory, Video]
// excludeGroupedVideo: 是否排除分组的视频
// page, pageSize: 分页
// sortBy: 排序字段，默认 create_time
// sort: 排序方向，默认 DESC
func (c *Client) ItemList(guid string, types []Type, excludeGroupedVideo bool, page, pageSize int, sortBy, sort string) ([]Item, int, error) {
	if types == nil {
		types = []Type{TypeMovie, TypeTV, TypeDirectory, TypeVideo}
	}
	if sortBy == "" {
		sortBy = "create_time"
	}
	if sort == "" {
		sort = "DESC"
	}
	typeFilter := map[string]interface{}{}
	if len(types) > 0 {
		ts := make([]string, 0, len(types))
		for _, t := range types {
			ts = append(ts, string(t))
		}
		typeFilter["type"] = ts
	}
	post := map[string]interface{}{
		"tags":        typeFilter,
		"sort_type":   sort,
		"sort_column": sortBy,
		"page":        page,
		"page_size":   pageSize,
	}
	if guid != "" {
		post["ancestor_guid"] = guid
	}
	if excludeGroupedVideo {
		post["exclude_grouped_video"] = 1
	}
	res, err := c.request("/item/list", "POST", nil, post, "", false)
	if err != nil || !res.Success() {
		return nil, 0, fmt.Errorf("item list failed: %v", err)
	}
	if res.Data == nil {
		return []Item{}, 0, nil
	}
	dataMap := res.DataMap()
	listRaw, ok := dataMap["list"]
	if !ok {
		return []Item{}, 0, nil
	}
	arr, err := asInterfaceList(listRaw)
	if err != nil {
		return []Item{}, 0, nil
	}
	items := make([]Item, 0, len(arr))
	for _, info := range arr {
		items = append(items, c.buildItem(info))
	}
	// total 为匹配条件下的总数（不受分页影响）
	return items, getInt(dataMap, "total"), nil
}

// Item 媒体详情
func (c *Client) Item(guid string) (*Item, error) {
	res, err := c.request(fmt.Sprintf("/item/%s", guid), "GET", nil, nil, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("item detail failed: %v", err)
	}
	if res.Data == nil {
		return nil, nil
	}
	item := c.buildItem(res.DataMap())
	return &item, nil
}

// ItemCount 获取指定媒体库的媒体条目总数
// types: 默认 [Movie, TV]
func (c *Client) ItemCount(guid string, types []Type) (int, error) {
	if types == nil {
		types = []Type{TypeMovie, TypeTV}
	}
	ts := make([]string, 0, len(types))
	for _, t := range types {
		ts = append(ts, string(t))
	}
	post := map[string]interface{}{
		"ancestor_guid":         guid,
		"tags":                  map[string]interface{}{"type": ts},
		"exclude_grouped_video": 1,
		"page":                  1,
		"page_size":             1,
	}
	res, err := c.request("/item/list", "POST", nil, post, "", false)
	if err != nil || !res.Success() {
		return 0, fmt.Errorf("item count failed: %v", err)
	}
	if res.Data == nil {
		return 0, nil
	}
	total := getInt(res.DataMap(), "total")
	if total == 0 {
		total = getInt(res.DataMap(), "total_count")
	}
	return total, nil
}

// SearchList 搜索影片、演员
func (c *Client) SearchList(keywords string) ([]Item, error) {
	res, err := c.request("/search/list", "GET", map[string]string{"q": keywords}, nil, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("search list failed: %v", err)
	}
	if res.Data == nil {
		return []Item{}, nil
	}
	arr, err := asInterfaceList(res.Data)
	if err != nil {
		return []Item{}, nil
	}
	items := make([]Item, 0, len(arr))
	for _, info := range arr {
		items = append(items, c.buildItem(info))
	}
	return items, nil
}

// DeleteItem 删除媒体
// deleteFile: true 删除媒体文件，false 仅从媒体库移除
func (c *Client) DeleteItem(guid string, deleteFile bool) (bool, error) {
	delFlag := 0
	if deleteFile {
		delFlag = 1
	}
	res, err := c.request(fmt.Sprintf("/item/%s", guid), "DELETE", nil, map[string]interface{}{
		"delete_file": delFlag,
		"media_guids": []interface{}{},
	}, "", false)
	if err != nil || !res.Success() {
		return false, fmt.Errorf("delete item failed: %v", err)
	}
	return res.Data != nil, nil
}

// SeasonList 查询季列表
func (c *Client) SeasonList(tvGuid string) ([]Item, error) {
	res, err := c.request(fmt.Sprintf("/season/list/%s", tvGuid), "GET", nil, nil, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("season list failed: %v", err)
	}
	if res.Data == nil {
		return []Item{}, nil
	}
	arr, err := asInterfaceList(res.Data)
	if err != nil {
		return []Item{}, nil
	}
	items := make([]Item, 0, len(arr))
	for _, info := range arr {
		items = append(items, c.buildItem(info))
	}
	return items, nil
}

// EpisodeList 查询剧集列表
func (c *Client) EpisodeList(seasonGuid string) ([]Item, error) {
	res, err := c.request(fmt.Sprintf("/episode/list/%s", seasonGuid), "GET", nil, nil, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("episode list failed: %v", err)
	}
	if res.Data == nil {
		return []Item{}, nil
	}
	arr, err := asInterfaceList(res.Data)
	if err != nil {
		return []Item{}, nil
	}
	items := make([]Item, 0, len(arr))
	for _, info := range arr {
		items = append(items, c.buildItem(info))
	}
	return items, nil
}

// PersonList 查询媒体演职员列表，返回 data.list
func (c *Client) PersonList(guid string) ([]Person, error) {
	res, err := c.request(fmt.Sprintf("/person/list/%s", guid), "POST", nil, map[string]interface{}{
		"page":      1,
		"page_size": 200,
	}, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("person list failed: %v", err)
	}
	if res.Data == nil {
		return []Person{}, nil
	}
	m := res.DataMap()
	if m == nil {
		return []Person{}, nil
	}
	listRaw, ok := m["list"]
	if !ok {
		return []Person{}, nil
	}
	arr, err := asInterfaceList(listRaw)
	if err != nil {
		return []Person{}, nil
	}
	persons := make([]Person, 0, len(arr))
	for _, info := range arr {
		p, err := c.buildPerson(info)
		if err != nil {
			continue
		}
		persons = append(persons, p)
	}
	return persons, nil
}

// buildPerson 从 map 构造 Person，处理 profile_path 相对路径
func (c *Client) buildPerson(info map[string]interface{}) (Person, error) {
	var p Person
	b, err := json.Marshal(info)
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return p, err
	}
	p.ProfilePath = c.buildImgAPIURL(p.ProfilePath)
	return p, nil
}

// PersonSearch 搜索演员，返回 data.list
func (c *Client) PersonSearch(keyword string, page, pageSize int) ([]PersonSearchResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 200
	}
	res, err := c.request("/person/search", "POST", nil, map[string]interface{}{
		"keyword":   keyword,
		"page":      page,
		"page_size": pageSize,
	}, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("person search failed: %v", err)
	}
	if res.Data == nil {
		return []PersonSearchResult{}, nil
	}
	m := res.DataMap()
	if m == nil {
		return []PersonSearchResult{}, nil
	}
	listRaw, ok := m["list"]
	if !ok {
		return []PersonSearchResult{}, nil
	}
	arr, err := asInterfaceList(listRaw)
	if err != nil {
		return []PersonSearchResult{}, nil
	}
	results := make([]PersonSearchResult, 0, len(arr))
	for _, info := range arr {
		r := PersonSearchResult{
			GUID:         getString(info, "guid"),
			Name:         getString(info, "name"),
			IMDBID:       getString(info, "imdbId"),
			TrimID:       getString(info, "trim_id"),
			IsOfficial:   getBool(info, "is_official"),
			OriginalName: getString(info, "original_name"),
			Profile:      c.buildImgAPIURL(getString(info, "profile")),
			IsFavorite:   getInt(info, "is_favorite"),
		}
		results = append(results, r)
	}
	return results, nil
}

// UploadImage 上传图片到飞牛临时存储
// imageData: 图片二进制数据
// filename: 文件名（含扩展名）
// imageType: 图片类型，如 "poster"
// 返回飞牛返回的 hash_path
func (c *Client) UploadImage(imageData []byte, filename, imageType string) (string, error) {
	if c.host == "" {
		return "", fmt.Errorf("host is empty")
	}

	apiPathStr := apiPath + "/image/temp/upload"
	reqURL := c.host + apiPathStr

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// image_type 字段
	if err := writer.WriteField("image_type", imageType); err != nil {
		return "", fmt.Errorf("write image_type field: %w", err)
	}

	// file 字段
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(imageData); err != nil {
		return "", fmt.Errorf("write file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequest("POST", reqURL, body)
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", c.host)
	req.Header.Set("Authorization", c.token)
	// multipart body 签名内容使用空字符串
	req.Header.Set("authx", c.getAuthx(apiPathStr, ""))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	var result RequestResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w, body: %s", err, string(respBody))
	}
	if !result.Success() {
		return "", fmt.Errorf("upload image failed: code=%d msg=%s", result.Code, result.Msg)
	}

	m := result.DataMap()
	if m == nil {
		return "", fmt.Errorf("upload image: empty data")
	}
	hashPath, ok := m["hash_path"].(string)
	if !ok || hashPath == "" {
		return "", fmt.Errorf("upload image: missing hash_path")
	}
	return hashPath, nil
}

// CreatePerson 在飞牛创建演员
// name: 演员名称
// profilePath: 演员头像路径（由 UploadImage 返回的 hash_path）
// 返回创建的演员 guid
func (c *Client) CreatePerson(name, profilePath string) (string, error) {
	res, err := c.request("/person/create", "POST", nil, map[string]interface{}{
		"name":         name,
		"profile_path": profilePath,
	}, "", false)
	if err != nil || !res.Success() {
		return "", fmt.Errorf("create person failed: %v, msg: %s", err, res.Msg)
	}
	m := res.DataMap()
	if m == nil {
		return "", fmt.Errorf("create person: empty data")
	}
	guid, ok := m["guid"].(string)
	if !ok || guid == "" {
		return "", fmt.Errorf("create person: missing guid")
	}
	return guid, nil
}
func (c *Client) PlayList() ([]Item, error) {
	res, err := c.request("/play/list", "GET", nil, nil, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("play list failed: %v", err)
	}
	if res.Data == nil {
		return []Item{}, nil
	}
	arr, err := asInterfaceList(res.Data)
	if err != nil {
		return []Item{}, nil
	}
	items := make([]Item, 0, len(arr))
	for _, info := range arr {
		items = append(items, c.buildItem(info))
	}
	return items, nil
}

// TaskRunning 当前是否有正在运行的任务
func (c *Client) TaskRunning() (bool, error) {
	res, err := c.request("/task/running", "GET", nil, nil, "", false)
	if err != nil || !res.Success() {
		return false, fmt.Errorf("task running failed: %v", err)
	}
	return res.Data != nil, nil
}

// GenreList 查询媒体类型列表
// lan: 语言，默认 zh-CN
func (c *Client) GenreList(lan string) ([]Genre, error) {
	if lan == "" {
		lan = "zh-CN"
	}
	res, err := c.request("/tag/genres", "GET", map[string]string{"lan": lan}, nil, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("genre list failed: %v", err)
	}
	arr, err := asInterfaceList(res.Data)
	if err != nil {
		return []Genre{}, nil
	}
	genres := make([]Genre, 0, len(arr))
	for _, info := range arr {
		g := Genre{
			ID:    getInt(info, "id"),
			Value: getString(info, "value"),
		}
		genres = append(genres, g)
	}
	return genres, nil
}

// BatchCreateGenres 批量新增自定义分类
// values: 分类名称列表
func (c *Client) BatchCreateGenres(values []string) ([]Genre, error) {
	if len(values) == 0 {
		return []Genre{}, nil
	}
	reqBody := map[string]interface{}{"values": values}
	res, err := c.request("/tag/custom/genres/batch", "POST", nil, reqBody, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("batch create genres failed: %v, msg: %s", err, res.Msg)
	}
	arr, err := asInterfaceList(res.Data)
	if err != nil {
		return []Genre{}, nil
	}
	genres := make([]Genre, 0, len(arr))
	for _, info := range arr {
		g := Genre{
			ID:    getInt(info, "id"),
			Value: getString(info, "value"),
		}
		genres = append(genres, g)
	}
	return genres, nil
}

// CountryList 查询国家地区列表
// lan: 语言，默认 zh-CN
func (c *Client) CountryList(lan string) ([]Country, error) {
	if lan == "" {
		lan = "zh-CN"
	}
	res, err := c.request("/tag/iso3166", "GET", map[string]string{"lan": lan}, nil, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("country list failed: %v", err)
	}
	arr, err := asInterfaceList(res.Data)
	if err != nil {
		return []Country{}, nil
	}
	countries := make([]Country, 0, len(arr))
	for _, info := range arr {
		c := Country{
			Key:   getString(info, "key"),
			Value: getString(info, "value"),
		}
		countries = append(countries, c)
	}
	return countries, nil
}

// EditDetail 获取媒体项编辑信息
func (c *Client) EditDetail(guid string) (*EditDetail, error) {
	res, err := c.request("/item/getEditDetail", "POST", nil, map[string]interface{}{
		"item_guid": guid,
	}, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("get edit detail failed: %v", err)
	}
	if res.Data == nil {
		return nil, nil
	}
	detail := &EditDetail{}
	b, _ := json.Marshal(res.DataMap())
	if err := json.Unmarshal(b, detail); err != nil {
		return nil, fmt.Errorf("unmarshal edit detail: %w", err)
	}
	return detail, nil
}

// SaveEditDetail 保存媒体项编辑信息
func (c *Client) SaveEditDetail(detail *EditDetail) (bool, error) {
	if detail == nil {
		return false, fmt.Errorf("nil edit detail")
	}
	b, err := json.Marshal(detail)
	if err != nil {
		return false, fmt.Errorf("marshal edit detail: %w", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(b, &body); err != nil {
		return false, fmt.Errorf("re-unmarshal edit detail: %w", err)
	}
	res, err := c.request("/item/saveEditDetail", "POST", nil, body, "", false)
	if err != nil || !res.Success() {
		return false, fmt.Errorf("save edit detail failed: %v", err)
	}
	return true, nil
}

// buildItem 从 map 构造 Item
func (c *Client) buildItem(info map[string]interface{}) Item {
	item := Item{}
	b, _ := json.Marshal(info)
	_ = json.Unmarshal(b, &item)
	item.Type = ParseType(getString(info, "type"))
	// 列表API: poster(单数); 详情API: posters(复数)
	poster := getString(info, "poster")
	if poster == "" {
		poster = getString(info, "posters")
	}
	item.Poster = c.buildImgAPIURL(poster)
	// 详情API: backdrops(复数), 列表API无此字段
	item.Backdrop = c.buildImgAPIURL(getString(info, "backdrops"))
	// 详情API: logos(复数), 列表API无此字段
	item.Logo = c.buildImgAPIURL(getString(info, "logos"))
	return item
}

// --- 辅助函数 ---

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case int64:
			return int(n)
		}
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getStringList(m map[string]interface{}, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, err := asStringList(v)
	if err != nil {
		return nil
	}
	return arr
}

// asStringList 尝试把 v 当作 []string 解析
func asStringList(v interface{}) ([]string, error) {
	if v == nil {
		return nil, fmt.Errorf("nil")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var arr []string
	if err := json.Unmarshal(b, &arr); err != nil {
		return nil, err
	}
	return arr, nil
}

// asInterfaceList 尝试把 v 当作 []map[string]interface{} 或 []interface{} 解析
func asInterfaceList(v interface{}) ([]map[string]interface{}, error) {
	if v == nil {
		return nil, fmt.Errorf("nil")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var arr []map[string]interface{}
	if err := json.Unmarshal(b, &arr); err != nil {
		return nil, err
	}
	return arr, nil
}
