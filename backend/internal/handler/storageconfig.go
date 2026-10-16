package handler

import (
	"context"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"gorm.io/gorm"

	"fntube/internal/model"
)

type StorageConfigHandler struct {
	db *gorm.DB
}

type storageMappingRequest struct {
	Name              string `json:"name"`
	GuangyaFolderID   string `json:"guangya_folder_id"`
	GuangyaFolderPath string `json:"guangya_folder_path"`
	SmartStrmTaskName string `json:"smartstrm_task_name"`
	StrmSavePath      string `json:"strm_save_path"`
}

func RegisterStorageConfigHandlers(h *server.Hertz, db *gorm.DB) {
	handler := &StorageConfigHandler{db: db}
	group := h.Group("/api/storage-config/mappings")
	group.GET("", handler.list)
	group.POST("", handler.create)
	group.PUT("/:id", handler.update)
	group.DELETE("/:id", handler.delete)
}

func (h *StorageConfigHandler) list(ctx context.Context, c *app.RequestContext) {
	var mappings []model.StorageMapping
	if err := h.db.Order("id asc").Find(&mappings).Error; err != nil {
		c.JSON(500, map[string]string{"error": "读取存储配置失败: " + err.Error()})
		return
	}
	c.JSON(200, mappings)
}

func (h *StorageConfigHandler) create(ctx context.Context, c *app.RequestContext) {
	request, ok := bindStorageMappingRequest(c)
	if !ok {
		return
	}

	mapping := model.StorageMapping{
		Name:              request.Name,
		GuangyaFolderID:   request.GuangyaFolderID,
		GuangyaFolderPath: request.GuangyaFolderPath,
		SmartStrmTaskName: request.SmartStrmTaskName,
		StrmSavePath:      request.StrmSavePath,
	}
	if err := h.db.Create(&mapping).Error; err != nil {
		c.JSON(500, map[string]string{"error": "创建存储配置失败: " + err.Error()})
		return
	}
	c.JSON(200, mapping)
}

func (h *StorageConfigHandler) update(ctx context.Context, c *app.RequestContext) {
	id, ok := storageMappingID(c)
	if !ok {
		return
	}
	request, ok := bindStorageMappingRequest(c)
	if !ok {
		return
	}

	var mapping model.StorageMapping
	if err := h.db.First(&mapping, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, map[string]string{"error": "存储配置不存在"})
			return
		}
		c.JSON(500, map[string]string{"error": "读取存储配置失败: " + err.Error()})
		return
	}

	if err := h.db.Model(&mapping).Updates(map[string]interface{}{
		"name":                 request.Name,
		"guangya_folder_id":    request.GuangyaFolderID,
		"guangya_folder_path":  request.GuangyaFolderPath,
		"smart_strm_task_name": request.SmartStrmTaskName,
		"strm_save_path":       request.StrmSavePath,
	}).Error; err != nil {
		c.JSON(500, map[string]string{"error": "更新存储配置失败: " + err.Error()})
		return
	}

	if err := h.db.First(&mapping, id).Error; err != nil {
		c.JSON(500, map[string]string{"error": "读取存储配置失败: " + err.Error()})
		return
	}
	c.JSON(200, mapping)
}

func (h *StorageConfigHandler) delete(ctx context.Context, c *app.RequestContext) {
	id, ok := storageMappingID(c)
	if !ok {
		return
	}
	result := h.db.Delete(&model.StorageMapping{}, id)
	if result.Error != nil {
		c.JSON(500, map[string]string{"error": "删除存储配置失败: " + result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(404, map[string]string{"error": "存储配置不存在"})
		return
	}
	c.JSON(200, map[string]string{"status": "ok"})
}

func bindStorageMappingRequest(c *app.RequestContext) (storageMappingRequest, bool) {
	var request storageMappingRequest
	if err := c.BindJSON(&request); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return request, false
	}

	request.Name = strings.TrimSpace(request.Name)
	request.GuangyaFolderID = strings.TrimSpace(request.GuangyaFolderID)
	request.GuangyaFolderPath = strings.TrimSpace(request.GuangyaFolderPath)
	request.SmartStrmTaskName = strings.TrimSpace(request.SmartStrmTaskName)
	request.StrmSavePath = strings.TrimSpace(request.StrmSavePath)

	switch {
	case request.Name == "":
		c.JSON(400, map[string]string{"error": "请填写名称"})
	case request.GuangyaFolderPath == "":
		c.JSON(400, map[string]string{"error": "请选择光鸭云盘目录"})
	case request.SmartStrmTaskName == "":
		c.JSON(400, map[string]string{"error": "请填写 SmartStrm 任务名"})
	case request.StrmSavePath == "":
		c.JSON(400, map[string]string{"error": "请填写 strm 保存目录"})
	default:
		return request, true
	}
	return request, false
}

func storageMappingID(c *app.RequestContext) (uint, bool) {
	value, err := strconv.ParseUint(string(c.Param("id")), 10, 64)
	if err != nil || value == 0 {
		c.JSON(400, map[string]string{"error": "无效的 id"})
		return 0, false
	}
	return uint(value), true
}
