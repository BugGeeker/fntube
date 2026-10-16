package handler

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"fntube/internal/fnos"
	"fntube/internal/systemtest"
)

type copyTestRequest struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type aclTestRequest struct {
	UID  int      `json:"uid"`
	Path []string `json:"path"`
}

type aclResult struct {
	Path      string `json:"path"`
	Readable  bool   `json:"readable"`
	Writable  bool   `json:"writable"`
	Deletable bool   `json:"deletable"`
}

func RegisterSystemTestHandlers(h *server.Hertz) {
	client := fnos.NewClient()

	h.POST("/api/system-test/copy", func(ctx context.Context, c *app.RequestContext) {
		var request copyTestRequest
		if err := c.BindJSON(&request); err != nil {
			c.JSON(400, map[string]string{"error": "请求参数无效"})
			return
		}
		result, err := systemtest.CopyPath(request.Source, request.Destination)
		if err != nil {
			c.JSON(400, map[string]string{"error": err.Error()})
			return
		}
		c.JSON(200, result)
	})

	h.GET("/api/system-test/shared-folders", func(ctx context.Context, c *app.RequestContext) {
		var result struct {
			Paths []string `json:"paths"`
		}
		if err := client.Call(ctx, "trim.file.getSharedAccessibleFolders", map[string]any{}, &result); err != nil {
			c.JSON(400, map[string]string{"error": err.Error()})
			return
		}
		if result.Paths == nil {
			result.Paths = []string{}
		}
		c.JSON(200, result)
	})

	h.POST("/api/system-test/acl", func(ctx context.Context, c *app.RequestContext) {
		var request aclTestRequest
		if err := c.BindJSON(&request); err != nil {
			c.JSON(400, map[string]string{"error": "请求参数无效"})
			return
		}
		if request.UID <= 0 {
			c.JSON(400, map[string]string{"error": "UID 必须大于 0"})
			return
		}
		paths := make([]string, 0, len(request.Path))
		for _, path := range request.Path {
			if path = strings.TrimSpace(path); path != "" {
				paths = append(paths, path)
			}
		}
		if len(paths) == 0 {
			c.JSON(400, map[string]string{"error": "至少填写一个检查路径"})
			return
		}

		var result []aclResult
		if err := client.Call(ctx, "trim.file.checkUserACL", map[string]any{
			"uid":  request.UID,
			"path": paths,
		}, &result); err != nil {
			c.JSON(400, map[string]string{"error": err.Error()})
			return
		}
		if result == nil {
			result = []aclResult{}
		}
		c.JSON(200, result)
	})
}
