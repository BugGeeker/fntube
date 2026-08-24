package trimmedia

import "strings"

// Category 媒体库分类
type Category string

const (
	CategoryMovie  Category = "Movie"
	CategoryTV     Category = "TV"
	CategoryMix    Category = "Mix"
	CategoryOthers Category = "Others"
)

// ParseCategory 将字符串转换为 Category，未知值返回 Others
func ParseCategory(s string) Category {
	switch Category(s) {
	case CategoryMovie, CategoryTV, CategoryMix:
		return Category(s)
	default:
		if strings.ToLower(s) == "music" || strings.ToLower(s) == "audio" {
			return Category(s)
		}
		return CategoryOthers
	}
}

// Type 媒体类型
type Type string

const (
	TypeMovie     Type = "Movie"
	TypeTV        Type = "TV"
	TypeSeason    Type = "Season"
	TypeEpisode   Type = "Episode"
	TypeVideo     Type = "Video"
	TypeDirectory Type = "Directory"
)

// ParseType 将字符串转换为 Type，未知值返回 Video
func ParseType(s string) Type {
	switch Type(s) {
	case TypeMovie, TypeTV, TypeSeason, TypeEpisode, TypeDirectory:
		return Type(s)
	default:
		return TypeVideo
	}
}

// User 飞牛用户
type User struct {
	GUID     string `json:"guid"`
	Username string `json:"username"`
	IsAdmin  int    `json:"is_admin"`
}

// MediaDb 媒体库
type MediaDb struct {
	GUID     string   `json:"guid"`
	Category Category `json:"category"`
	Name     string   `json:"name"`
	Posters  []string `json:"posters"`
	DirList  []string `json:"dir_list"`
}

// MediaDbSummary 媒体数量统计
type MediaDbSummary struct {
	Favorite int `json:"favorite"`
	Movie    int `json:"movie"`
	TV       int `json:"tv"`
	Video    int `json:"video"`
	Total    int `json:"total"`
}

// Version 飞牛影视版本
type Version struct {
	Frontend string `json:"frontend"`
	Backend  string `json:"backend"`
}

// Genre 媒体类型
type Genre struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
}

// Country 国家地区
type Country struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Item 媒体项
type Item struct {
	GUID                  string      `json:"guid"`
	AncestorGUID          string      `json:"ancestor_guid"`
	Type                  Type        `json:"type"`
	TVTitle               string      `json:"tv_title"`
	ParentTitle           string      `json:"parent_title"`
	Title                 string      `json:"title"`
	OriginalTitle         string      `json:"original_title"`
	Overview              string      `json:"overview"`
	Logo                  string      `json:"logo"`
	Poster                string      `json:"poster"`
	Backdrop              string      `json:"backdrop"`
	DoubanID              int         `json:"douban_id"`
	IMDBID                string      `json:"imdb_id"`
	TrimID                string      `json:"trim_id"`
	ReleaseDate           string      `json:"release_date"`
	AirDate               string      `json:"air_date"`
	VoteAverage           string      `json:"vote_average"`
	SeasonNumber          int         `json:"season_number"`
	EpisodeNumber         int         `json:"episode_number"`
	Duration              int         `json:"duration"` // 片长(秒)
	Ts                    int         `json:"ts"`       // 已播放(秒)
	Watched               int         `json:"watched"`  // 1:已看完
	PosterWidth           int         `json:"poster_width"`
	PosterHeight          int         `json:"poster_height"`
	Genres                []int       `json:"genres"`
	Runtime               int         `json:"runtime"` // 片长(分钟)
	ProductionCountries   []string    `json:"production_countries"`
	IsFavorite            int         `json:"is_favorite"`
	IsWatched             int         `json:"is_watched"`
	WatchedTs             int64       `json:"watched_ts"`
	NumberOfEpisodes      int         `json:"number_of_episodes"`
	LocalNumberOfEpisodes int         `json:"local_number_of_episodes"`
	LocalNumberOfSeasons  int         `json:"local_number_of_seasons"`
	CanPlay               int         `json:"can_play"`
	PlayError             string      `json:"play_error"`
	ParentGuid            string      `json:"parent_guid"`
	AncestorName          string      `json:"ancestor_name"`
	AncestorCategory      string      `json:"ancestor_category"`
	PlayItemGuid          string      `json:"play_item_guid"`
	LogicType             int         `json:"logic_type"`
	MediaStream           MediaStream `json:"media_stream"`
}

// MediaStream 媒体流信息
type MediaStream struct {
	Resolutions    []string `json:"resolutions"`
	AudioType      []string `json:"audio_type"`
	ColorRangeType []string `json:"color_range_type"`
}

// Person 演职员
type Person struct {
	ItemGUID           string `json:"item_guid"`
	PersonGUID         string `json:"person_guid"`
	Role               string `json:"role"`
	Job                string `json:"job"`
	Order              int    `json:"order"`
	Department         string `json:"department"`
	TrimID             string `json:"trim_id"`
	IMDBID             string `json:"imdb_id"`
	TmdbID             int    `json:"tmdb_id"`
	Lan                string `json:"lan"`
	Name               string `json:"name"`
	OriginalName       string `json:"original_name"`
	AlsoKnowAs         string `json:"also_know_as"`
	Biography          string `json:"biography"`
	KnownForDepartment string `json:"known_for_department"`
	ProfilePath        string `json:"profile_path"`
	Gender             int    `json:"gender"`
}

// PersonSearchResult 演员搜索结果
type PersonSearchResult struct {
	GUID          string `json:"guid"`
	Name          string `json:"name"`
	IMDBID        string `json:"imdbId"`
	TrimID        string `json:"trim_id"`
	IsOfficial    bool   `json:"is_official"`
	OriginalName  string `json:"original_name"`
	Profile       string `json:"profile"`
	IsFavorite    int    `json:"is_favorite"`
}

// TmdbID 从 trim_id 提取 tmdb id
// 飞牛给 tmdbid 加了前缀用以区分 tv 或 movie：tt* 或 tm*
func (i *Item) TmdbID() int {
	if i.TrimID == "" {
		return 0
	}
	if strings.HasPrefix(i.TrimID, "tt") || strings.HasPrefix(i.TrimID, "tm") {
		var n int
		for _, c := range i.TrimID[2:] {
			if c < '0' || c > '9' {
				return 0
			}
			n = n*10 + int(c-'0')
		}
		return n
	}
	return 0
}

// ItemListResult /item/list 接口返回结构
type ItemListResult struct {
	List  []map[string]interface{} `json:"list"`
	Total int                      `json:"total"`
}

// EditCredit 编辑信息中的演职员条目
type EditCredit struct {
	PersonGUID  string `json:"person_guid"`
	Name        string `json:"name"`
	Job         string `json:"job"`
	Role        string `json:"role"`
	Order       int    `json:"order"`
	ProfilePath string `json:"profile_path"`
}

// EditDetail /item/getEditDetail 返回的媒体编辑信息
type EditDetail struct {
	ItemGUID                  string        `json:"item_guid"`
	TrimID                    string        `json:"trim_id"`
	IsOfficial                bool          `json:"is_official"`
	Title                     string        `json:"title"`
	TitleLocked               bool          `json:"title_locked"`
	Overview                  string        `json:"overview"`
	OverviewLocked            bool          `json:"overview_locked"`
	Rating                    float64       `json:"rating"`
	RatingLocked              bool          `json:"rating_locked"`
	AirDate                   string        `json:"air_date"`
	AirDateLocked             bool          `json:"air_date_locked"`
	FirstAirDate              interface{}   `json:"first_air_date"`
	FirstAirDateLocked        bool          `json:"first_air_date_locked"`
	LastAirDate               interface{}   `json:"last_air_date"`
	LastAirDateLocked         bool          `json:"last_air_date_locked"`
	ContentRating             string        `json:"content_rating"`
	ContentRatingLocked       bool          `json:"content_rating_locked"`
	Backdrops                 string        `json:"backdrops"`
	BackdropsLocked           bool          `json:"backdrops_locked"`
	Logos                     string        `json:"logos"`
	LogosLocked               bool          `json:"logos_locked"`
	Posters                   string        `json:"posters"`
	PostersLocked             bool          `json:"posters_locked"`
	PosterType                int           `json:"poster_type"`
	GenresLocked              bool          `json:"genres_locked"`
	Genres                    []int         `json:"genres"`
	ProductionCountries       []string      `json:"production_countries"`
	ProductionCountriesLocked bool          `json:"production_countries_locked"`
	Credits                   []EditCredit  `json:"credits"`
	CreditsLocked             bool          `json:"credits_locked"`
	JobTypesOpts              []string      `json:"job_types_opts"`
	ContentRatingOpts         []string      `json:"content_rating_opts"`
}
