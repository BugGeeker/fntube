package fnos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultSocketPath = "/var/run/trim_open_gateway_apiscope.socket"
	apiPath           = "/api/v1/trimapp"
	appName           = "FnTube"
)

type Client struct {
	httpClient *http.Client
	token      func() string
}

type requestEnvelope struct {
	ReqID   string `json:"reqId"`
	Req     string `json:"req"`
	AppName string `json:"appName"`
	Data    any    `json:"data"`
}

type responseEnvelope struct {
	ReqID string          `json:"reqId"`
	Code  int             `json:"code"`
	Msg   string          `json:"msg"`
	Data  json.RawMessage `json:"data"`
}

func NewClient() *Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", defaultSocketPath)
		},
	}
	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second,
		},
		token: func() string {
			return strings.TrimSpace(os.Getenv("TRIM_API_TOKEN"))
		},
	}
}

func (c *Client) Call(ctx context.Context, method string, data any, result any) error {
	token := c.token()
	if token == "" {
		return errors.New("当前环境未提供 TRIM_API_TOKEN，请在飞牛系统中测试")
	}

	body, err := json.Marshal(requestEnvelope{
		ReqID:   fmt.Sprintf("%d", time.Now().UnixNano()),
		Req:     method,
		AppName: appName,
		Data:    data,
	})
	if err != nil {
		return fmt.Errorf("编码飞牛 API 请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost"+apiPath, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建飞牛 API 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("调用飞牛 API 失败: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取飞牛 API 响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("飞牛 API 返回 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var envelope responseEnvelope
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return fmt.Errorf("解析飞牛 API 响应失败: %w", err)
	}
	if envelope.Code != 0 {
		return fmt.Errorf("飞牛 API 返回错误 code=%d: %s", envelope.Code, envelope.Msg)
	}
	if result == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, result); err != nil {
		return fmt.Errorf("解析飞牛 API 数据失败: %w", err)
	}
	return nil
}
