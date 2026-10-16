package smartstrm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type Version struct {
	Version         string  `json:"version"`
	RemoteVersion   string  `json:"remote_version"`
	IsNew           bool    `json:"is_new"`
	Notify          string  `json:"notify"`
	LicenseStoreURL *string `json:"license_store_url"`
	ChangeLogURL    string  `json:"change_log_url"`
}

type Task struct {
	Name    string                 `json:"name"`
	Config  map[string]interface{} `json:"config"`
	NextRun interface{}            `json:"next_run"`
	Running bool                   `json:"running"`
}

func TestConnection(
	ctx context.Context,
	baseURL string,
	username string,
	password string,
) (Version, error) {
	client, err := login(ctx, baseURL, username, password)
	if err != nil {
		return Version{}, err
	}

	var version Version
	if err := getJSON(ctx, client, endpoint(baseURL, "/api/version"), &version); err != nil {
		return version, fmt.Errorf("查询版本失败: %w", err)
	}
	if version.Version == "" {
		return version, fmt.Errorf("版本接口未返回当前版本")
	}
	return version, nil
}

func FetchTasks(
	ctx context.Context,
	baseURL string,
	username string,
	password string,
) (map[string]Task, error) {
	client, err := login(ctx, baseURL, username, password)
	if err != nil {
		return nil, err
	}

	var tasks map[string]Task
	if err := getJSON(ctx, client, endpoint(baseURL, "/api/tasks"), &tasks); err != nil {
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}
	return tasks, nil
}

func RunTask(
	ctx context.Context,
	baseURL string,
	username string,
	password string,
	taskName string,
	storagePath string,
) error {
	client, err := login(ctx, baseURL, username, password)
	if err != nil {
		return err
	}

	body, err := json.Marshal(map[string]string{"storage_path": storagePath})
	if err != nil {
		return fmt.Errorf("创建任务参数失败: %w", err)
	}
	runURL := endpoint(baseURL, "/api/tasks/"+url.PathEscape(taskName)+"/run")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, runURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建任务执行请求失败: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("执行任务失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("执行任务失败，服务返回状态码 %d", response.StatusCode)
	}
	return nil
}

func getJSON(ctx context.Context, client *http.Client, requestURL string, target interface{}) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("服务返回状态码 %d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}
	return nil
}

func login(
	ctx context.Context,
	baseURL string,
	username string,
	password string,
) (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("创建登录会话失败: %w", err)
	}
	client := &http.Client{
		Timeout: 15 * time.Second,
		Jar:     jar,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	form := url.Values{
		"username": {username},
		"password": {password},
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint(baseURL, "/login"),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("创建登录请求失败: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("登录请求失败: %w", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("登录失败，服务返回状态码 %d", response.StatusCode)
	}
	return client, nil
}

func endpoint(baseURL, suffix string) string {
	return strings.TrimRight(baseURL, "/") + suffix
}
