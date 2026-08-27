package trimmedia

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	apiKey     = "16CCEB3D-AB42-077D-36A1-F355324E4237"
	secret     = "NDzZTVxnRKP8Z0jXg1VAMonaG8akvh"
	apiPath    = "/api/v1"
	userAgent  = "fntube/1.0"
	timeFormat = "2006-01-02 15:04:05"
)

// Client 飞牛影视 API 客户端
type Client struct {
	host       string
	token      string
	accessCode string
	httpClient *http.Client
	version    *Version
}

// NewClient 创建飞牛影视 API 客户端
func NewClient(host, accessCode string) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		host:       strings.TrimRight(host, "/"),
		accessCode: accessCode,
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 15 * time.Second,
		},
	}
}

// Host 返回服务端地址
func (c *Client) Host() string {
	return c.host
}

// Token 返回当前 token
func (c *Client) Token() string {
	return c.token
}

// Version 返回飞牛版本
func (c *Client) Version() *Version {
	return c.version
}

// Close 关闭客户端会话
func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}

// VerifyAccessCode 校验访问码
// 未配置访问码或校验通过返回 true
func (c *Client) VerifyAccessCode() bool {
	if c.accessCode == "" {
		return true
	}
	root := c.host
	if strings.HasSuffix(root, "/v") {
		root = root[:len(root)-len("/v")]
	}
	code := url.QueryEscape(c.accessCode)
	reqURL := fmt.Sprintf("%s/c/%s", root, code)
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		log.Printf("[trimmedia] 校验访问码失败，无法访问 %s: %v", reqURL, err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[trimmedia] 飞牛访问码校验失败，请检查访问码是否正确")
		return false
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("[trimmedia] 飞牛访问码校验失败，状态码：%d", resp.StatusCode)
		return false
	}
	return true
}

// getAuthx 计算消息签名
func (c *Client) getAuthx(apiPath, body string) string {
	if !strings.HasPrefix(apiPath, "/v") {
		apiPath = "/v" + apiPath
	}
	nonce := strconv.Itoa(100000 + rand.Intn(900000))
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	dataHash := md5Hex(body)
	sign := md5Hex(strings.Join([]string{secret, apiPath, nonce, ts, dataHash, apiKey}, "_"))
	return fmt.Sprintf("nonce=%s&timestamp=%s&sign=%s", nonce, ts, sign)
}

// RequestResult 统一响应结构
type RequestResult struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// DataMap 将 Data 断言为 map[string]interface{}，失败返回 nil
func (r *RequestResult) DataMap() map[string]interface{} {
	if m, ok := r.Data.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// Success code==0 表示成功
func (r *RequestResult) Success() bool {
	return r.Code == 0
}

// request 发送请求的通用方法
// method: GET/POST/DELETE，为空时根据 data 是否为 nil 决定
// params: query 参数
// data: JSON body
// basePath: 路径前缀覆盖（如 /api/v2）
// suppressLog: 是否禁止日志
func (c *Client) request(api, method string, params map[string]string, data interface{}, basePath string, suppressLog bool) (*RequestResult, error) {
	if c.host == "" || api == "" {
		return nil, fmt.Errorf("host or api is empty")
	}
	prefix := basePath
	if prefix == "" {
		prefix = apiPath
	}
	var apiPathStr string
	if !strings.HasPrefix(api, "/") {
		apiPathStr = prefix + "/" + api
	} else {
		apiPathStr = prefix + api
	}
	reqURL := c.host + apiPathStr

	if method == "" {
		if data == nil {
			method = "GET"
		} else {
			method = "POST"
		}
	}
	method = strings.ToUpper(method)

	var jsonBody string
	var bodyReader io.Reader
	if method != "GET" {
		if data != nil {
			b, err := json.Marshal(data)
			if err != nil {
				return nil, fmt.Errorf("marshal body: %w", err)
			}
			jsonBody = string(b)
			bodyReader = bytes.NewReader(b)
		}
	}

	// 构造 query string
	if params != nil {
		q := url.Values{}
		for k, v := range params {
			q.Set(k, v)
		}
		if strings.Contains(reqURL, "?") {
			reqURL += "&" + q.Encode()
		} else {
			reqURL += "?" + q.Encode()
		}
	}

	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", c.host)
	req.Header.Set("Authorization", c.token)
	// 签名使用 body 或 query string
	signContent := jsonBody
	if signContent == "" && params != nil {
		q := url.Values{}
		for k, v := range params {
			q.Set(k, v)
		}
		signContent = q.Encode()
	}
	req.Header.Set("authx", c.getAuthx(apiPathStr, signContent))
	if jsonBody != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if !suppressLog {
			log.Printf("[trimmedia] 请求接口 %s 异常: %v", reqURL, err)
		}
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if !suppressLog {
			log.Printf("[trimmedia] 读取响应 %s 失败: %v", reqURL, err)
		}
		return nil, fmt.Errorf("read body: %w", err)
	}
	if len(body) == 0 {
		if !suppressLog {
			log.Printf("[trimmedia] 请求接口 %s 失败: 空响应", reqURL)
		}
		return nil, fmt.Errorf("empty response from %s", reqURL)
	}

	var result RequestResult
	if err := json.Unmarshal(body, &result); err != nil {
		if !suppressLog {
			log.Printf("[trimmedia] 解析响应 %s 失败: %v, body: %s", reqURL, err, string(body))
		}
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if result.Code != 0 && !suppressLog {
		log.Printf("[trimmedia] 请求接口 %s 失败，错误码: %d %s", reqURL, result.Code, result.Msg)
	}
	return &result, nil
}

// SysVersion 获取飞牛影视版本号
func (c *Client) SysVersion() (*Version, error) {
	res, err := c.request("/sys/version", "GET", nil, nil, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("sys version failed: %v", err)
	}
	if res.Data == nil {
		return nil, nil
	}
	ver := &Version{}
	if v, ok := res.DataMap()["version"].(string); ok {
		ver.Frontend = v
	}
	if v, ok := res.DataMap()["mediasrvVersion"].(string); ok {
		ver.Backend = v
	}
	c.version = ver
	return ver, nil
}

// Login 登录飞牛影视
// 优先使用 v2 协议登录（密码传输 SHA256 摘要），v2 不存在时回退 v1 明文登录
func (c *Client) Login(username, password string) (string, error) {
	// 开启访问码后需先通过访问码校验
	if !c.VerifyAccessCode() {
		return "", fmt.Errorf("access code verification failed")
	}

	// v2 协议登录
	pwdHash := sha256Hex(password)
	res, err := c.request("/user/loginByPassword", "POST", nil, map[string]interface{}{
		"username":  username,
		"password":  pwdHash,
		"app_name":  "trimemedia-web",
	}, "/api/v2", true)

	if err == nil && res != nil {
		if res.Success() {
			if token, ok := res.DataMap()["token"].(string); ok {
				c.token = token
				return token, nil
			}
		} else {
			// v2 接口存在但登录失败（如账号密码错误）
			return "", fmt.Errorf("login failed: code=%d msg=%s", res.Code, res.Msg)
		}
	}

	// v2 不可用（旧版服务端），回退 v1 明文登录
	res2, err2 := c.request("/login", "POST", nil, map[string]interface{}{
		"username": username,
		"password": password,
		"app_name": "trimemedia-web",
	}, "", false)
	if err2 != nil {
		return "", fmt.Errorf("v1 login error: %w", err2)
	}
	if !res2.Success() {
		return "", fmt.Errorf("v1 login failed: code=%d msg=%s", res2.Code, res2.Msg)
	}
	if token, ok := res2.DataMap()["token"].(string); ok {
		c.token = token
		return token, nil
	}
	return "", fmt.Errorf("login: no token in response")
}

// Logout 退出账号
func (c *Client) Logout() bool {
	if c.token == "" {
		return true
	}
	res, err := c.request("/user/logout", "POST", nil, nil, "", false)
	if err != nil || !res.Success() {
		return false
	}
	c.token = ""
	return true
}

// UserInfo 当前用户信息
func (c *Client) UserInfo() (*User, error) {
	res, err := c.request("/user/info", "GET", nil, nil, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("user info failed: %v", err)
	}
	if res.Data == nil {
		return nil, nil
	}
	u := &User{}
	if v, ok := res.DataMap()["guid"].(string); ok {
		u.GUID = v
	}
	if v, ok := res.DataMap()["username"].(string); ok {
		u.Username = v
	}
	if v, ok := res.DataMap()["is_admin"].(float64); ok {
		u.IsAdmin = int(v)
	}
	return u, nil
}

// UserList 用户列表(仅管理员有权访问)
func (c *Client) UserList() ([]User, error) {
	res, err := c.request("/manager/user/list", "GET", nil, nil, "", false)
	if err != nil || !res.Success() {
		return nil, fmt.Errorf("user list failed: %v", err)
	}
	// data 可能是 list
	dataBytes, _ := json.Marshal(res.Data)
	var list []map[string]interface{}
	if err := json.Unmarshal(dataBytes, &list); err != nil {
		return nil, fmt.Errorf("user list unmarshal: %w", err)
	}
	users := make([]User, 0, len(list))
	for _, info := range list {
		u := User{}
		if v, ok := info["guid"].(string); ok {
			u.GUID = v
		}
		if v, ok := info["username"].(string); ok {
			u.Username = v
		}
		if v, ok := info["is_admin"].(float64); ok {
			u.IsAdmin = int(v)
		}
		users = append(users, u)
	}
	return users, nil
}

// buildImgAPIURL 构建图片 API URL
func (c *Client) buildImgAPIURL(imgPath string) string {
	if imgPath == "" {
		return ""
	}
	if !strings.HasPrefix(imgPath, "/") {
		imgPath = "/" + imgPath
	}
	return fmt.Sprintf("%s/sys/img%s", apiPath, imgPath)
}

// ProxyImage 代理获取飞牛图片，携带登录会话 Cookie 与鉴权头访问
// path 为相对路径，形如 /api/v1/sys/img/...（可含 ?w= 尺寸参数）
func (c *Client) ProxyImage(path string) ([]byte, string, error) {
	if path == "" {
		return nil, "", fmt.Errorf("empty image path")
	}
	imageURL, err := url.Parse(path)
	if err != nil {
		return nil, "", fmt.Errorf("parse image path: %w", err)
	}
	// 飞牛部分图片经 w 参数缩放后会返回无法解码的 WebP，统一获取原图。
	imageURL.RawQuery = ""
	reqURL := c.host + imageURL.String()
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("new image request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "image/*,*/*;q=0.8")
	req.Header.Set("Referer", c.host)
	req.Header.Set("Authorization", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("proxy image request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("proxy image status: %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read image body: %w", err)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	return data, contentType, nil
}

// --- 辅助函数 ---

func md5Hex(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func sha256Hex(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
