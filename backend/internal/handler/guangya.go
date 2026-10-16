package handler

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"gorm.io/gorm"

	"fntube/internal/guangya"
)

type GuangyaHandler struct {
	service *guangya.Service
}

func RegisterGuangyaHandlers(h *server.Hertz, db *gorm.DB) {
	service := guangya.NewService(db)
	service.StartCloudJobWorker()
	hd := &GuangyaHandler{service: service}
	g := h.Group("/api/guangya")
	g.GET("/config", hd.getConfig)
	g.POST("/unbind", hd.unbind)
	g.POST("/qr/start", hd.startQR)
	g.POST("/qr/poll", hd.pollQR)
	g.POST("/sms/send", hd.sendSMS)
	g.POST("/sms/verify", hd.verifySMS)
	g.POST("/cloud/resolve", hd.resolveCloudURL)
	g.GET("/cloud/folders", hd.listCloudFolders)
	g.POST("/cloud/folders", hd.createCloudFolder)
	g.POST("/cloud/task", hd.createCloudTask)
	g.GET("/cloud/jobs", hd.listCloudJobs)
	g.DELETE("/cloud/jobs/:id", hd.deleteCloudJob)
	g.GET("/cloud/jobs/events", hd.streamCloudJobs)
}

func (h *GuangyaHandler) getConfig(ctx context.Context, c *app.RequestContext) {
	config, err := h.service.Config()
	if err != nil {
		c.JSON(500, map[string]string{"error": "读取光鸭云盘配置失败: " + err.Error()})
		return
	}
	c.JSON(200, config)
}

func (h *GuangyaHandler) unbind(ctx context.Context, c *app.RequestContext) {
	if err := h.service.Unbind(); err != nil {
		c.JSON(500, map[string]string{"error": "解绑失败: " + err.Error()})
		return
	}
	c.JSON(200, map[string]string{"status": "ok"})
}

func (h *GuangyaHandler) startQR(ctx context.Context, c *app.RequestContext) {
	result, err := h.service.StartQR()
	if err != nil {
		c.JSON(400, map[string]string{"error": "获取扫码授权信息失败: " + err.Error()})
		return
	}
	c.JSON(200, result)
}

func (h *GuangyaHandler) pollQR(ctx context.Context, c *app.RequestContext) {
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": "请求参数无效"})
		return
	}
	result, err := h.service.PollQR(req.SessionID)
	if err != nil {
		status := 400
		if !errors.Is(err, guangya.ErrSessionNotFound) && !errors.Is(err, guangya.ErrSessionExpired) {
			status = 502
		}
		c.JSON(status, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, result)
}

func (h *GuangyaHandler) sendSMS(ctx context.Context, c *app.RequestContext) {
	var req struct {
		Phone        string `json:"phone"`
		CaptchaToken string `json:"captcha_token"`
		SessionID    string `json:"session_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": "请求参数无效"})
		return
	}
	result, err := h.service.SendSMS(req.Phone, req.CaptchaToken, req.SessionID)
	if err != nil {
		c.JSON(400, map[string]string{"error": "发送短信验证码失败: " + err.Error()})
		return
	}
	c.JSON(200, result)
}

func (h *GuangyaHandler) verifySMS(ctx context.Context, c *app.RequestContext) {
	var req struct {
		SessionID string `json:"session_id"`
		Code      string `json:"code"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": "请求参数无效"})
		return
	}
	config, err := h.service.VerifySMS(req.SessionID, req.Code)
	if err != nil {
		c.JSON(400, map[string]string{"error": "绑定失败: " + err.Error()})
		return
	}
	c.JSON(200, config)
}

func (h *GuangyaHandler) resolveCloudURL(ctx context.Context, c *app.RequestContext) {
	var req struct {
		URL string `json:"url"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": "请求参数无效"})
		return
	}
	result, err := h.service.ResolveURL(req.URL)
	if err != nil {
		status := 502
		if errors.Is(err, guangya.ErrNotBound) {
			status = 400
		}
		c.JSON(status, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, result)
}

func (h *GuangyaHandler) listCloudFolders(ctx context.Context, c *app.RequestContext) {
	parentID := string(c.Query("parent_id"))
	page, _ := strconv.Atoi(string(c.Query("page")))
	pageSize, _ := strconv.Atoi(string(c.Query("page_size")))
	result, err := h.service.Folders(parentID, page, pageSize)
	if err != nil {
		status := 502
		if errors.Is(err, guangya.ErrNotBound) {
			status = 400
		}
		c.JSON(status, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, result)
}

func (h *GuangyaHandler) createCloudFolder(ctx context.Context, c *app.RequestContext) {
	var req struct {
		Name     string `json:"name"`
		ParentID string `json:"parent_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": "请求参数无效"})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		c.JSON(400, map[string]string{"error": "文件夹名称不能为空"})
		return
	}
	result, err := h.service.CreateFolder(req.Name, req.ParentID)
	if err != nil {
		status := 502
		if errors.Is(err, guangya.ErrNotBound) {
			status = 400
		}
		c.JSON(status, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, result)
}

func (h *GuangyaHandler) createCloudTask(ctx context.Context, c *app.RequestContext) {
	var req struct {
		URL                 string                      `json:"url"`
		ParentID            string                      `json:"parent_id"`
		NewName             string                      `json:"new_name"`
		FileIndexes         []int                       `json:"file_indexes"`
		ResolvedIsDirectory bool                        `json:"resolved_is_directory"`
		SelectedFiles       []guangya.SelectedCloudFile `json:"selected_files"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": "请求参数无效"})
		return
	}
	result, err := h.service.CreateCloudTask(guangya.CreateCloudTaskInput{
		FileIndexes:         req.FileIndexes,
		URL:                 req.URL,
		ParentID:            req.ParentID,
		NewName:             req.NewName,
		ResolvedIsDirectory: req.ResolvedIsDirectory,
		SelectedFiles:       req.SelectedFiles,
	})
	if err != nil {
		status := 502
		if errors.Is(err, guangya.ErrNotBound) {
			status = 400
		}
		c.JSON(status, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, result)
}

func (h *GuangyaHandler) listCloudJobs(ctx context.Context, c *app.RequestContext) {
	page, _ := strconv.Atoi(string(c.Query("page")))
	pageSize, _ := strconv.Atoi(string(c.Query("page_size")))
	result, err := h.service.CloudAddJobs(page, pageSize)
	if err != nil {
		c.JSON(500, map[string]string{"error": "读取云添加记录失败: " + err.Error()})
		return
	}
	c.JSON(200, result)
}

func (h *GuangyaHandler) deleteCloudJob(ctx context.Context, c *app.RequestContext) {
	id, err := strconv.ParseUint(string(c.Param("id")), 10, 64)
	if err != nil || id == 0 {
		c.JSON(400, map[string]string{"error": "云添加记录 ID 无效"})
		return
	}
	if err := h.service.DeleteCloudAddJob(uint(id)); err != nil {
		if errors.Is(err, guangya.ErrCloudJobNotFound) {
			c.JSON(404, map[string]string{"error": err.Error()})
			return
		}
		c.JSON(500, map[string]string{"error": "删除云添加记录失败: " + err.Error()})
		return
	}
	c.JSON(200, map[string]string{"status": "ok"})
}

func (h *GuangyaHandler) streamCloudJobs(ctx context.Context, c *app.RequestContext) {
	events, unsubscribe := h.service.SubscribeCloudJobs()
	defer unsubscribe()

	writer := sse.NewWriter(c)
	defer writer.Close()

	active, err := h.service.ActiveCloudAddJobs()
	if err != nil {
		_ = writer.WriteEvent("", "error", []byte(`{"error":"读取进行中任务失败"}`))
		return
	}
	for _, job := range active {
		if err := writeCloudJobEvent(writer, job); err != nil {
			return
		}
	}

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-events:
			if event.Type == "deleted" {
				if err := writeCloudJobDeletedEvent(writer, event.ID); err != nil {
					return
				}
			} else if err := writeCloudJobEvent(writer, event.Job); err != nil {
				return
			}
		case <-heartbeat.C:
			if err := writer.WriteKeepAlive(); err != nil {
				return
			}
		}
	}
}

func writeCloudJobEvent(writer *sse.Writer, job any) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return writer.WriteEvent("", "cloud-add-progress", data)
}

func writeCloudJobDeletedEvent(writer *sse.Writer, id uint) error {
	data, err := json.Marshal(map[string]uint{"id": id})
	if err != nil {
		return err
	}
	return writer.WriteEvent("", "cloud-add-deleted", data)
}
