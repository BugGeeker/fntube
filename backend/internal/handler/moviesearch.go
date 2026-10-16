package handler

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"gorm.io/gorm"

	"fntube/internal/moviesearch"
)

type MovieSearchHandler struct {
	db      *gorm.DB
	service *moviesearch.Service
}

type movieSearchConfigView struct {
	Source       string `json:"source"`
	JavDBDomain  string `json:"javdb_domain"`
	JavBusDomain string `json:"javbus_domain"`
}

func RegisterMovieSearchHandlers(h *server.Hertz, db *gorm.DB) {
	handler := &MovieSearchHandler{db: db, service: moviesearch.NewDefaultService()}
	group := h.Group("/api/movie-search")
	group.GET("/sources", handler.getSources)
	group.GET("/config", handler.getConfig)
	group.POST("/config", handler.saveConfig)
	group.GET("/search", handler.search)
}

func (h *MovieSearchHandler) getSources(ctx context.Context, c *app.RequestContext) {
	c.JSON(200, map[string]interface{}{"data": h.service.Sources()})
}

func (h *MovieSearchHandler) getConfig(ctx context.Context, c *app.RequestContext) {
	config := h.loadConfig()
	c.JSON(200, config)
}

func (h *MovieSearchHandler) loadConfig() movieSearchConfigView {
	var config movieSearchConfigView
	_ = h.db.Table("movie_search_configs").Order("id desc").First(&config).Error
	if !h.service.HasSource(config.Source) {
		config.Source = moviesearch.DefaultSource
	}
	if config.JavDBDomain == "" {
		config.JavDBDomain = moviesearch.DefaultJavDBDomain
	}
	if config.JavBusDomain == "" {
		config.JavBusDomain = moviesearch.DefaultJavBusDomain
	}
	return config
}

func (h *MovieSearchHandler) saveConfig(ctx context.Context, c *app.RequestContext) {
	var request movieSearchConfigView
	if err := c.BindJSON(&request); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}
	current := h.loadConfig()
	if strings.TrimSpace(request.Source) == "" {
		request.Source = current.Source
	}
	if strings.TrimSpace(request.JavDBDomain) == "" {
		request.JavDBDomain = current.JavDBDomain
	}
	if strings.TrimSpace(request.JavBusDomain) == "" {
		request.JavBusDomain = current.JavBusDomain
	}

	request.Source = strings.ToLower(strings.TrimSpace(request.Source))
	if !h.service.HasSource(request.Source) {
		c.JSON(400, map[string]string{"error": "不支持的搜索源"})
		return
	}
	var err error
	request.JavDBDomain, err = moviesearch.NormalizeDomain(request.JavDBDomain)
	if err != nil {
		c.JSON(400, map[string]string{"error": "JavDB " + err.Error()})
		return
	}
	request.JavBusDomain, err = moviesearch.NormalizeDomain(request.JavBusDomain)
	if err != nil {
		c.JSON(400, map[string]string{"error": "JavBus " + err.Error()})
		return
	}

	var id int64
	row := h.db.Table("movie_search_configs").Select("id").Order("id desc").Limit(1).Row()
	_ = row.Scan(&id)
	values := map[string]interface{}{
		"source":         request.Source,
		"jav_db_domain":  request.JavDBDomain,
		"jav_bus_domain": request.JavBusDomain,
	}
	if id > 0 {
		err = h.db.Table("movie_search_configs").Where("id = ?", id).Updates(values).Error
	} else {
		err = h.db.Table("movie_search_configs").Create(values).Error
	}
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, map[string]interface{}{"status": "ok", "config": request})
}

func (h *MovieSearchHandler) search(ctx context.Context, c *app.RequestContext) {
	source := string(c.Query("source"))
	keyword := string(c.Query("q"))
	if strings.TrimSpace(source) == "" {
		c.JSON(400, map[string]string{"error": "搜索源不能为空"})
		return
	}
	if strings.TrimSpace(keyword) == "" {
		c.JSON(400, map[string]string{"error": "搜索关键词不能为空"})
		return
	}

	config := h.loadConfig()
	service := moviesearch.NewConfiguredService(config.JavBusDomain, config.JavDBDomain)
	results, err := service.Search(ctx, source, keyword)
	if err != nil {
		c.JSON(502, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, map[string]interface{}{"data": results})
}
