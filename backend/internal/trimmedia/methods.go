package trimmedia

import (
	"encoding/json"
	"fmt"
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
func (c *Client) ItemList(guid string, types []Type, excludeGroupedVideo bool, page, pageSize int, sortBy, sort string) ([]Item, error) {
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
		return nil, fmt.Errorf("item list failed: %v", err)
	}
	if res.Data == nil {
		return []Item{}, nil
	}
	listRaw, ok := res.DataMap()["list"]
	if !ok {
		return []Item{}, nil
	}
	arr, err := asInterfaceList(listRaw)
	if err != nil {
		return []Item{}, nil
	}
	items := make([]Item, 0, len(arr))
	for _, info := range arr {
		items = append(items, c.buildItem(info))
	}
	return items, nil
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
		"delete_file":  delFlag,
		"media_guids":  []interface{}{},
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

// PlayList 继续观看列表
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

// buildItem 从 map 构造 Item
func (c *Client) buildItem(info map[string]interface{}) Item {
	item := Item{}
	b, _ := json.Marshal(info)
	_ = json.Unmarshal(b, &item)
	item.Type = ParseType(getString(info, "type"))
	item.Posters = c.buildImgAPIURL(getString(info, "posters"))
	item.Backdrops = c.buildImgAPIURL(getString(info, "backdrops"))
	item.Logos = c.buildImgAPIURL(getString(info, "logos"))
	poster := getString(info, "poster")
	if poster != "" {
		item.Poster = c.buildImgAPIURL(poster)
	} else {
		item.Poster = item.Posters
	}
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
