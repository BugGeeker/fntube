package metatube

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client MetaTube 服务端 API 客户端
// 依据 Apifox「metatube」项目 (8742733) 与 metatube-sdk-go 官方路由实现。
type Client struct {
	host       string
	token      string
	httpClient *http.Client
}

// NewClient 创建 MetaTube API 客户端
func NewClient(host, token string) *Client {
	return &Client{
		host:       strings.TrimRight(strings.TrimSpace(host), "/"),
		token:      token,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// Version MetaTube 服务端版本信息（GET / 返回 data 字段）
type Version struct {
	App     string `json:"app"`
	Version string `json:"version"`
}

// versionResp GET / 响应结构
// MetaTube 服务端统一响应格式：{"data": ..., "error": {"code": .., "message": ..}}
type versionResp struct {
	Data  Version `json:"data"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Version 获取服务端版本信息（GET /）
func (c *Client) Version() (*Version, error) {
	if c.host == "" {
		return nil, fmt.Errorf("服务地址为空")
	}

	req, err := http.NewRequest(http.MethodGet, c.host+"/", nil)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("无法连接 MetaTube 服务 %s: %w", c.host, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("响应状态码 %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var vr versionResp
	if err := json.Unmarshal(body, &vr); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if vr.Error != nil {
		msg := vr.Error.Message
		if msg == "" {
			msg = http.StatusText(vr.Error.Code)
		}
		return nil, fmt.Errorf("服务端返回错误: %s", msg)
	}

	return &vr.Data, nil
}

// VerifyToken 验证 Token 是否有效。
// 未配置 token 时视为通过；配置了 token 时通过私有接口 GET /v1/db/version
// （该接口需 Authorization: Bearer <token>）校验，401 表示 token 无效。
func (c *Client) VerifyToken() bool {
	if c.token == "" {
		return true
	}

	req, err := http.NewRequest(http.MethodGet, c.host+"/v1/db/version", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// Close 关闭空闲连接
func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}