package trimmedia

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// Service 飞牛影视服务层
type Service struct {
	host          string
	username      string
	password      string
	accessCode    string
	playHost      string
	syncLibraries []string

	client    *Client
	userInfo  *User
	version   *Version
	libraries map[string]MediaDb
}

// NewService 从配置创建服务（不自动连接）
func NewService(host, username, password, accessCode, playHost string, syncLibraries []string) *Service {
	return &Service{
		host:          strings.TrimRight(host, "/"),
		username:      username,
		password:      password,
		accessCode:    accessCode,
		playHost:      strings.TrimRight(playHost, "/"),
		syncLibraries: syncLibraries,
		libraries:     map[string]MediaDb{},
	}
}

// IsConfigured 配置是否完整
func (s *Service) IsConfigured() bool {
	return s.host != "" && s.username != "" && s.password != ""
}

// IsAuthenticated 是否已登录
func (s *Service) IsAuthenticated() bool {
	return s.IsConfigured() && s.client != nil && s.client.token != "" && s.userInfo != nil
}

// IsInactive 是否需要重连
func (s *Service) IsInactive() bool {
	if !s.IsAuthenticated() {
		return true
	}
	var err error
	s.userInfo, err = s.client.UserInfo()
	return err != nil || s.userInfo == nil
}

// Client 返回底层 API 客户端
func (s *Service) Client() *Client {
	return s.client
}

// createAPI 创建 API 客户端并验证版本
func createAPI(host, accessCode string) (*Client, *Version, error) {
	host = strings.TrimRight(host, "/")
	if host == "" {
		return nil, nil, fmt.Errorf("host is empty")
	}
	// 如果不以 /v 结尾，尝试补上
	if !strings.HasSuffix(host, "/v") {
		c := NewClient(host+"/v", accessCode)
		if c.VerifyAccessCode() {
			if ver, err := c.SysVersion(); err == nil && ver != nil {
				return c, ver, nil
			}
		}
		c.Close()
	}
	// 测试用户配置的地址
	c := NewClient(host, accessCode)
	if c.VerifyAccessCode() {
		if ver, err := c.SysVersion(); err == nil && ver != nil {
			return c, ver, nil
		}
	}
	c.Close()
	return nil, nil, fmt.Errorf("无法连接飞牛影视 %s", host)
}

// Reconnect 重连
func (s *Service) Reconnect() bool {
	if !s.IsConfigured() {
		return false
	}
	s.Disconnect()

	client, ver, err := createAPI(s.host, s.accessCode)
	if err != nil {
		log.Printf("[trimmedia] %v", err)
		return false
	}
	s.client = client
	s.version = ver

	token, err := s.client.Login(s.username, s.password)
	if err != nil {
		log.Printf("[trimmedia] 登录失败: %v", err)
		return false
	}
	if token == "" {
		return false
	}
	userInfo, err := s.client.UserInfo()
	if err != nil || userInfo == nil {
		return false
	}
	s.userInfo = userInfo
	log.Printf("[trimmedia] %s 成功登录飞牛影视", s.username)
	// 刷新媒体库列表
	s.cacheLibraries()
	return true
}

// cacheLibraries 缓存媒体库列表
func (s *Service) cacheLibraries() {
	if !s.IsAuthenticated() {
		return
	}
	var list []MediaDb
	var err error
	if s.userInfo != nil && s.userInfo.IsAdmin == 1 {
		list, err = s.client.MdbList()
	} else {
		list, err = s.client.MediaDbList()
	}
	if err != nil {
		log.Printf("[trimmedia] 获取媒体库列表失败: %v", err)
		return
	}
	s.libraries = map[string]MediaDb{}
	for _, lib := range list {
		s.libraries[lib.GUID] = lib
	}
}

// UpdateConfig 更新配置并立即重连（用于保存配置后热更新，无需重启）
func (s *Service) UpdateConfig(host, username, password, accessCode, playHost, syncLibraries string) bool {
	s.host = strings.TrimRight(host, "/")
	s.username = username
	s.password = password
	s.accessCode = accessCode
	s.playHost = strings.TrimRight(playHost, "/")
	s.syncLibraries = LoadSyncLibraries(syncLibraries)
	return s.Reconnect()
}

// Disconnect 断开连接
func (s *Service) Disconnect() {
	if s.client != nil {
		s.client.Logout()
		s.client.Close()
		s.client = nil
		s.userInfo = nil
		log.Printf("[trimmedia] %s 已断开飞牛影视", s.username)
	}
}

// GetLibraries 获取媒体服务器所有媒体库列表
// hidden=true 时过滤掉被屏蔽的库
func (s *Service) GetLibraries(hidden bool) ([]Library, error) {
	if !s.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}
	var list []MediaDb
	var err error
	if s.userInfo != nil && s.userInfo.IsAdmin == 1 {
		list, err = s.client.MdbList()
	} else {
		list, err = s.client.MediaDbList()
	}
	if err != nil {
		return nil, err
	}
	s.libraries = map[string]MediaDb{}
	for _, lib := range list {
		s.libraries[lib.GUID] = lib
	}
	libraries := make([]Library, 0, len(list))
	for _, lib := range list {
		// 类型映射
		libType := "other"
		switch lib.Category {
		case CategoryMovie:
			libType = "movie"
		case CategoryTV:
			libType = "tv"
		case CategoryMix:
			libType = "mix"
		case CategoryOthers:
			libType = "other"
		}
		if strings.ToLower(string(lib.Category)) == "music" || strings.ToLower(string(lib.Category)) == "audio" {
			libType = "music"
		}
		// item 数量（包含 Video 类型以统计 Others 分类中的视频）
		count, _ := s.client.ItemCount(lib.GUID, []Type{TypeMovie, TypeTV, TypeVideo})
		// 图片为相对路径，由前端经 /api/trimmedia/img 代理加载（携带飞牛 Cookie）
		imgList := make([]string, 0, len(lib.Posters))
		for _, p := range lib.Posters {
			imgList = append(imgList, fmt.Sprintf("%s?w=256", p))
		}
		playHost := s.playHost
		if playHost == "" {
			playHost = s.client.Host()
		}
		libraries = append(libraries, Library{
			ID:        lib.GUID,
			Name:      lib.Name,
			Type:      libType,
			Path:      lib.DirList,
			ItemCount: count,
			ImageList: imgList,
			Link:      fmt.Sprintf("%s/library/%s", playHost, lib.GUID),
		})
	}
	return libraries, nil
}

// GetItems 获取媒体项目列表，支持递归展开目录
// parent: 媒体库ID
// startIndex: 起始页（从1开始）
// limit: 每页数量，-1 表示全部
func (s *Service) GetItems(parent string, startIndex, limit int) ([]MediaServerItem, error) {
	if !s.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}
	pageSize := limit
	if pageSize < 0 {
		pageSize = -1
	}
	items, err := s.client.ItemList(parent, []Type{TypeMovie, TypeTV, TypeVideo, TypeDirectory}, true, startIndex+1, pageSize, "create_time", "DESC")
	if err != nil {
		return nil, err
	}
	result := make([]MediaServerItem, 0, len(items))
	for _, item := range items {
		if item.Type == TypeDirectory {
			// 递归展开目录
			sub, err := s.GetItems(item.GUID, 0, limit)
			if err == nil {
				result = append(result, sub...)
			}
		} else if item.Type == TypeMovie || item.Type == TypeTV || item.Type == TypeVideo {
			result = append(result, s.buildMediaServerItem(item))
		}
	}
	return result, nil
}

// GetItemInfo 获取单个项目详情
func (s *Service) GetItemInfo(itemID string) (*MediaServerItem, error) {
	if !s.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}
	item, err := s.client.Item(itemID)
	if err != nil || item == nil {
		return nil, err
	}
	result := s.buildMediaServerItem(*item)
	return &result, nil
}

// GetPersonList 获取媒体演职员列表
func (s *Service) GetPersonList(itemID string) ([]Person, error) {
	if !s.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}
	if s.client == nil {
		return nil, fmt.Errorf("no client")
	}
	return s.client.PersonList(itemID)
}

// GetTVEpisodes 根据剧 ID 获取季和集列表
// season 指定时只返回该季
func (s *Service) GetTVEpisodes(itemID, title, year string, season *int) (string, map[int][]int, error) {
	if !s.IsAuthenticated() {
		return "", nil, fmt.Errorf("not authenticated")
	}
	if itemID == "" {
		// 按标题搜索
		items, err := s.client.SearchList(title)
		if err != nil {
			return "", nil, err
		}
		for _, item := range items {
			if item.Type != TypeTV {
				continue
			}
			if title == item.Title || title == item.OriginalTitle {
				if year == "" || (item.AirDate != "" && item.AirDate[:4] == year) {
					itemID = item.GUID
					break
				}
			}
		}
		if itemID == "" {
			return "", nil, nil
		}
	}
	item, err := s.client.Item(itemID)
	if err != nil || item == nil {
		return "", nil, fmt.Errorf("item not found: %v", err)
	}
	seasons, err := s.client.SeasonList(itemID)
	if err != nil {
		return "", nil, err
	}
	if len(seasons) == 0 {
		return "", nil, nil
	}
	if season != nil {
		filtered := make([]Item, 0, 1)
		for _, s := range seasons {
			if s.SeasonNumber == *season {
				filtered = append(filtered, s)
				break
			}
		}
		seasons = filtered
	}
	seasonEpisodes := map[int][]int{}
	for _, seasonItem := range seasons {
		episodes, err := s.client.EpisodeList(seasonItem.GUID)
		if err != nil {
			continue
		}
		for _, ep := range episodes {
			seasonEpisodes[ep.SeasonNumber] = append(seasonEpisodes[ep.SeasonNumber], ep.EpisodeNumber)
		}
	}
	return itemID, seasonEpisodes, nil
}

// GetPlayURL 获取媒体播放链接
func (s *Service) GetPlayURL(itemID string) (string, error) {
	if !s.IsAuthenticated() {
		return "", fmt.Errorf("not authenticated")
	}
	item, err := s.client.Item(itemID)
	if err != nil || item == nil {
		return "", err
	}
	host := s.playHost
	if host == "" {
		host = s.client.Host()
	}
	return buildPlayURL(host, *item), nil
}

// GetResume 获取继续观看列表
func (s *Service) GetResume(num int) ([]PlayItem, error) {
	if !s.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}
	items, err := s.client.PlayList()
	if err != nil {
		return nil, err
	}
	result := make([]PlayItem, 0, num)
	for _, item := range items {
		if num > 0 && len(result) >= num {
			break
		}
		result = append(result, s.buildPlayItem(item))
	}
	return result, nil
}

// GetLatest 获取最近更新列表
func (s *Service) GetLatest(num int) ([]PlayItem, error) {
	if !s.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}
	pageSize := num * 5
	if pageSize < 100 {
		pageSize = 100
	}
	items, err := s.client.ItemList("", []Type{TypeMovie, TypeTV}, true, 1, pageSize, "create_time", "DESC")
	if err != nil {
		return nil, err
	}
	result := make([]PlayItem, 0, num)
	for _, item := range items {
		if num > 0 && len(result) >= num {
			break
		}
		result = append(result, s.buildPlayItem(item))
	}
	return result, nil
}

// GetStatistics 获取媒体数量统计
func (s *Service) GetStatistics() (*MediaDbSummary, error) {
	if !s.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}
	return s.client.MediaDbSum()
}

// RefreshLibrary 按路径刷新所在媒体库
// items: 路径列表
func (s *Service) RefreshLibrary(items []string) (bool, error) {
	if !s.IsAuthenticated() {
		return false, fmt.Errorf("not authenticated")
	}
	if s.userInfo == nil || s.userInfo.IsAdmin != 1 {
		return false, fmt.Errorf("飞牛仅支持管理员账号刷新媒体库")
	}
	libSet := map[string]bool{}
	for _, path := range items {
		libGUID := s.matchLibraryByPath(path)
		if libGUID == "" {
			// 有匹配失败，刷新整个库
			return s.RefreshRootLibrary()
		}
		libSet[libGUID] = true
	}
	// 必须调用否则容易误报 Task duplicate
	_, _ = s.client.TaskRunning()
	for guid := range libSet {
		lib, ok := s.libraries[guid]
		if !ok {
			return s.RefreshRootLibrary()
		}
		log.Printf("[trimmedia] 刷新媒体库：%s", lib.Name)
		ok, err := s.client.MdbScan(guid)
		if err != nil || !ok {
			return s.RefreshRootLibrary()
		}
	}
	return true, nil
}

// RefreshRootLibrary 刷新所有媒体库
func (s *Service) RefreshRootLibrary() (bool, error) {
	if !s.IsAuthenticated() {
		return false, fmt.Errorf("not authenticated")
	}
	if s.userInfo == nil || s.userInfo.IsAdmin != 1 {
		return false, fmt.Errorf("飞牛仅支持管理员账号刷新媒体库")
	}
	_, _ = s.client.TaskRunning()
	log.Printf("[trimmedia] 刷新所有媒体库")
	return s.client.MdbScanAll()
}

// Search 搜索
func (s *Service) Search(keyword string) ([]MediaServerItem, error) {
	if !s.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}
	items, err := s.client.SearchList(keyword)
	if err != nil {
		return nil, err
	}
	result := make([]MediaServerItem, 0, len(items))
	for _, item := range items {
		result = append(result, s.buildMediaServerItem(item))
	}
	return result, nil
}

// matchLibraryByPath 按路径匹配媒体库
func (s *Service) matchLibraryByPath(path string) string {
	if path == "" {
		return ""
	}
	path = strings.TrimSpace(path)
	for _, lib := range s.libraries {
		for _, d := range lib.DirList {
			d = strings.TrimSpace(d)
			if d == "" {
				continue
			}
			if isSubPath(path, d) {
				return lib.GUID
			}
		}
	}
	return ""
}

// buildMediaServerItem 从 Item 构造统一的媒体项
func (s *Service) buildMediaServerItem(item Item) MediaServerItem {
	return MediaServerItem{
		Guid:                  item.GUID,
		ImdbID:                item.IMDBID,
		TrimID:                item.TrimID,
		TVTitle:               item.TVTitle,
		ParentTitle:           item.ParentTitle,
		Title:                 item.Title,
		Logos:                 item.Logos,
		OriginalTitle:         item.OriginalTitle,
		Backdrops:             item.Backdrops,
		Posters:               item.Posters,
		PosterWidth:           item.PosterWidth,
		PosterHeight:          item.PosterHeight,
		VoteAverage:           item.VoteAverage,
		Genres:                item.Genres,
		ReleaseDate:           item.ReleaseDate,
		Runtime:               item.Runtime,
		ProductionCountries:   item.ProductionCountries,
		Overview:              item.Overview,
		IsFavorite:            item.IsFavorite,
		IsWatched:             item.IsWatched,
		WatchedTs:             item.WatchedTs,
		AirDate:               item.AirDate,
		SeasonNumber:          item.SeasonNumber,
		NumberOfEpisodes:      item.NumberOfEpisodes,
		LocalNumberOfEpisodes: item.LocalNumberOfEpisodes,
		LocalNumberOfSeasons:  item.LocalNumberOfSeasons,
		CanPlay:               item.CanPlay,
		Type:                  string(item.Type),
		PlayError:             item.PlayError,
		ParentGuid:            item.ParentGuid,
		AncestorGuid:          item.AncestorGUID,
		AncestorName:          item.AncestorName,
		AncestorCategory:      item.AncestorCategory,
		PlayItemGuid:          item.PlayItemGuid,
		Duration:              item.Duration,
		LogicType:             item.LogicType,
		MediaStream:           item.MediaStream,
	}
}

// buildPlayItem 从 Item 构造播放项
func (s *Service) buildPlayItem(item Item) PlayItem {
	var title, subtitle, typ string
	if item.Type == TypeEpisode {
		title = item.TVTitle
		subtitle = fmt.Sprintf("S%d:%d - %s", item.SeasonNumber, item.EpisodeNumber, item.Title)
	} else {
		title = item.Title
		if item.Type == TypeMovie {
			subtitle = "电影"
		} else {
			subtitle = "视频"
		}
	}
	if item.Type == TypeMovie || item.Type == TypeVideo {
		typ = "movie"
	} else {
		typ = "tv"
	}
	host := s.playHost
	if host == "" {
		host = s.client.Host()
	}
	imageURL := item.Poster
	var percent float64
	if item.Duration > 0 && item.Ts > 0 {
		percent = float64(item.Ts) / float64(item.Duration) * 100
	}
	return PlayItem{
		ID:       item.GUID,
		Title:    title,
		Subtitle: subtitle,
		Type:     typ,
		Image:    imageURL,
		Link:     buildPlayURL(host, item),
		Percent:  percent,
	}
}

// buildPlayURL 拼装播放链接
func buildPlayURL(host string, item Item) string {
	switch item.Type {
	case TypeEpisode:
		return fmt.Sprintf("%s/tv/episode/%s", host, item.GUID)
	case TypeSeason:
		return fmt.Sprintf("%s/tv/season/%s", host, item.GUID)
	case TypeMovie:
		return fmt.Sprintf("%s/movie/%s", host, item.GUID)
	case TypeTV:
		return fmt.Sprintf("%s/tv/%s", host, item.GUID)
	default:
		return fmt.Sprintf("%s/other/%s", host, item.GUID)
	}
}

// isSubPath 判断 child 是否是 parent 的子路径
func isSubPath(child, parent string) bool {
	child = strings.TrimSpace(child)
	parent = strings.TrimSpace(parent)
	if child == parent {
		return true
	}
	return strings.HasPrefix(child, parent+"/") || strings.HasPrefix(child, parent+"\\")
}

// contains 字符串切片是否包含
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// --- 对外使用的 DTO 结构 ---

// Library 媒体库信息
type Library struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Path      []string `json:"path"`
	ItemCount int      `json:"item_count"`
	ImageList []string `json:"image_list"`
	Link      string   `json:"link"`
}

// MediaServerItem 统一媒体项，对齐 /item 接口响应字段
type MediaServerItem struct {
	Guid                  string      `json:"guid"`
	ImdbID                string      `json:"imdb_id"`
	TrimID                string      `json:"trim_id"`
	TVTitle               string      `json:"tv_title"`
	ParentTitle           string      `json:"parent_title"`
	Title                 string      `json:"title"`
	Logos                 string      `json:"logos"`
	OriginalTitle         string      `json:"original_title"`
	Backdrops             string      `json:"backdrops"`
	Posters               string      `json:"posters"`
	PosterWidth           int         `json:"poster_width"`
	PosterHeight          int         `json:"poster_height"`
	VoteAverage           string      `json:"vote_average"`
	Genres                []int       `json:"genres"`
	ReleaseDate           string      `json:"release_date"`
	Runtime               int         `json:"runtime"`
	ProductionCountries   []string    `json:"production_countries"`
	Overview              string      `json:"overview"`
	IsFavorite            int         `json:"is_favorite"`
	IsWatched             int         `json:"is_watched"`
	WatchedTs             int64       `json:"watched_ts"`
	AirDate               string      `json:"air_date"`
	SeasonNumber          int         `json:"season_number"`
	NumberOfEpisodes      int         `json:"number_of_episodes"`
	LocalNumberOfEpisodes int         `json:"local_number_of_episodes"`
	LocalNumberOfSeasons  int         `json:"local_number_of_seasons"`
	CanPlay               int         `json:"can_play"`
	Type                  string      `json:"type"`
	PlayError             string      `json:"play_error"`
	ParentGuid            string      `json:"parent_guid"`
	AncestorGuid          string      `json:"ancestor_guid"`
	AncestorName          string      `json:"ancestor_name"`
	AncestorCategory      string      `json:"ancestor_category"`
	PlayItemGuid          string      `json:"play_item_guid"`
	Duration              int         `json:"duration"`
	LogicType             int         `json:"logic_type"`
	MediaStream           MediaStream `json:"media_stream"`
}

// PlayItem 播放项
type PlayItem struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Subtitle string  `json:"subtitle"`
	Type     string  `json:"type"`
	Image    string  `json:"image"`
	Link     string  `json:"link"`
	Percent  float64 `json:"percent"`
}

// LoadSyncLibraries 从 JSON 字符串解析同步媒体库列表
func LoadSyncLibraries(s string) []string {
	if s == "" {
		return nil
	}
	var list []string
	if err := json.Unmarshal([]byte(s), &list); err != nil {
		return nil
	}
	return list
}
