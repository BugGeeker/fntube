package guangya

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	guangyaclient "github.com/BugGeeker/guangyaclient-go"
	"gorm.io/gorm"

	"fntube/internal/model"
	"fntube/internal/smartstrm"
)

const (
	loginSessionTTL  = 10 * time.Minute
	defaultQRScope   = "user"
	defaultSMSTarget = "ANY"
	tokenRefreshSkew = 30 * time.Second
)

var (
	ErrSessionNotFound = errors.New("登录会话不存在，请重新发起绑定")
	ErrSessionExpired  = errors.New("登录会话已过期，请重新发起绑定")
)

type authClient interface {
	DeviceCode(scope string) (guangyaclient.DeviceCodeResponse, error)
	DeviceToken(deviceCode string) (guangyaclient.TokenResponse, error)
	LoginSMSInit(phone, captchaToken string) (guangyaclient.CaptchaInitResponse, error)
	LoginSMSSend(phone, captchaToken, target string) (guangyaclient.SMSVerificationResponse, error)
	LoginSMSVerify(verificationID, verificationCode string) (guangyaclient.SMSVerifyResponse, error)
	LoginSMSSignin(verificationCode, verificationToken, username, captchaToken string) (guangyaclient.TokenResponse, error)
	UserInfo() (guangyaclient.UserInfoResponse, error)
	Credentials() credentials
}

type cloudClient interface {
	CloudResolveURL(url string) (guangyaclient.CloudResolveURLResponse, error)
	CloudTaskList(page, pageSize int, status []int, taskIDs []string) (guangyaclient.CloudTaskListResponse, error)
	FSFiles(parentID any, page, pageSize, orderBy, sortType int, fileTypes []int, resType, dirType *int, needPlayRecord bool) (guangyaclient.FileListResponse, error)
	FSCreateDir(dirName string, parentID any, failIfNameExist bool) (guangyaclient.FSCreateDirResponse, error)
	FSMove(fileIDs []any, parentID any) (guangyaclient.FileTaskResponse, error)
	FSRename(fileID, newName string) (guangyaclient.EmptyResponse, error)
	GetTaskStatus(taskID string) (guangyaclient.TaskStatusResponse, error)
	CloudCreateTask(fileIndexes []int, url string, parentID any, newName string) (guangyaclient.CloudCreateTaskResponse, error)
	RefreshToken(refresh string) (guangyaclient.TokenResponse, error)
	Credentials() credentials
}

type credentials struct {
	AccessToken  string
	RefreshToken string
	DeviceID     string
	ExpiresAt    time.Time
}

type sdkClient struct {
	*guangyaclient.Client
}

func (c *sdkClient) RefreshToken(refresh string) (guangyaclient.TokenResponse, error) {
	// The SDK checks expiry for every request, including its refresh request.
	// Clear the stale deadline to prevent RefreshToken from recursively refreshing itself.
	c.TokenExpiresAt = time.Time{}
	return c.Client.RefreshToken(refresh)
}

func (c *sdkClient) Credentials() credentials {
	return credentials{
		AccessToken:  c.Token,
		RefreshToken: c.RefreshTokenValue,
		DeviceID:     c.DeviceID,
		ExpiresAt:    c.TokenExpiresAt,
	}
}

type clientFactory func() authClient
type cloudClientFactory func(credentials) cloudClient

type loginSession struct {
	mu             sync.Mutex
	client         authClient
	expiresAt      time.Time
	deviceCode     string
	phone          string
	captchaToken   string
	verificationID string
}

type ConfigView struct {
	Bound          bool       `json:"bound"`
	Subject        string     `json:"subject,omitempty"`
	Name           string     `json:"name,omitempty"`
	PhoneNumber    string     `json:"phone_number,omitempty"`
	LoginMethod    string     `json:"login_method,omitempty"`
	TokenExpiresAt *time.Time `json:"token_expires_at,omitempty"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

type QRStartResult struct {
	SessionID               string    `json:"session_id"`
	UserCode                string    `json:"user_code"`
	VerificationURL         string    `json:"verification_url"`
	VerificationURIComplete string    `json:"verification_uri_complete"`
	Interval                int       `json:"interval"`
	ExpiresAt               time.Time `json:"expires_at"`
}

type QRPollResult struct {
	State  string      `json:"state"`
	Config *ConfigView `json:"config,omitempty"`
}

type SMSSendResult struct {
	SessionID       string `json:"session_id,omitempty"`
	CaptchaRequired bool   `json:"captcha_required"`
	CaptchaURL      string `json:"captcha_url,omitempty"`
}

type Service struct {
	db             *gorm.DB
	newClient      clientFactory
	newCloud       cloudClientFactory
	now            func() time.Time
	mu             sync.Mutex
	tokenMu        sync.Mutex
	sessions       map[string]*loginSession
	workerOnce     sync.Once
	subscriberMu   sync.Mutex
	subscribers    map[uint64]chan CloudJobEvent
	nextSubscriber uint64
	runSmartStrm   func(model.SmartStrmConfig, string, string) error
}

func NewService(db *gorm.DB) *Service {
	return newService(db, func() authClient {
		return &sdkClient{Client: guangyaclient.NewClient("", "", "")}
	})
}

func newService(db *gorm.DB, factory clientFactory) *Service {
	return &Service{
		db:        db,
		newClient: factory,
		newCloud: func(creds credentials) cloudClient {
			client := guangyaclient.NewClient(creds.AccessToken, creds.RefreshToken, creds.DeviceID)
			client.TokenExpiresAt = creds.ExpiresAt
			return &sdkClient{Client: client}
		},
		now:         time.Now,
		sessions:    make(map[string]*loginSession),
		subscribers: make(map[uint64]chan CloudJobEvent),
		runSmartStrm: func(config model.SmartStrmConfig, taskName, storagePath string) error {
			return smartstrm.RunTask(
				context.Background(),
				config.URL,
				config.Username,
				config.Password,
				taskName,
				storagePath,
			)
		},
	}
}

var ErrNotBound = errors.New("请先绑定光鸭云盘账号")
var ErrCloudJobNotFound = errors.New("云添加记录不存在")

type CloudFile struct {
	FileName  string      `json:"fileName"`
	FileSize  int64       `json:"fileSize,omitempty"`
	FileIndex *int        `json:"fileIndex,omitempty"`
	IsDir     bool        `json:"isDir,omitempty"`
	Subfiles  []CloudFile `json:"subfiles,omitempty"`
}

type CloudResolved struct {
	ResType  int         `json:"resType"`
	URL      string      `json:"url"`
	InfoHash string      `json:"infoHash,omitempty"`
	FileName string      `json:"fileName,omitempty"`
	FileSize int64       `json:"fileSize,omitempty"`
	Subfiles []CloudFile `json:"subfiles,omitempty"`
}

type CloudFolder struct {
	FileID   string `json:"fileId"`
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize,omitempty"`
	ResType  int    `json:"resType"`
	ParentID string `json:"parentId,omitempty"`
}

type CloudFolderList struct {
	List  []CloudFolder `json:"list"`
	Total int           `json:"total"`
}

type CloudTask struct {
	TaskID string `json:"taskId"`
	URL    string `json:"url"`
}

type SelectedCloudFile struct {
	FileIndex int      `json:"fileIndex"`
	FileName  string   `json:"fileName"`
	FileSize  int64    `json:"fileSize,omitempty"`
	Path      []string `json:"path,omitempty"`
}

type CreateCloudTaskInput struct {
	FileIndexes          []int
	URL                  string
	ParentID             string
	NewName              string
	StorageMappingID     uint
	ResolvedIsDirectory  bool
	SelectedFiles        []SelectedCloudFile
	SmartStrmTaskName    string
	SmartStrmStorageRoot string
	StrmSavePath         string
}

func (s *Service) cloudConfig() (model.GuangyaConfig, error) {
	var cfg model.GuangyaConfig
	err := s.db.Order("id desc").First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || strings.TrimSpace(cfg.AccessToken) == "" {
		return model.GuangyaConfig{}, ErrNotBound
	}
	if err != nil {
		return model.GuangyaConfig{}, err
	}
	return cfg, nil
}

func (s *Service) cloud(cfg model.GuangyaConfig) cloudClient {
	var expiresAt time.Time
	if cfg.TokenExpiresAt != nil {
		expiresAt = *cfg.TokenExpiresAt
	}
	return s.newCloud(credentials{
		AccessToken:  cfg.AccessToken,
		RefreshToken: cfg.RefreshToken,
		DeviceID:     cfg.DeviceID,
		ExpiresAt:    expiresAt,
	})
}

func cloudOperation[T any](s *Service, operation func(cloudClient) (T, error)) (T, error) {
	var zero T

	// Serialize refresh and persistence so concurrent requests cannot overwrite newer credentials.
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()

	cfg, err := s.cloudConfig()
	if err != nil {
		return zero, err
	}
	client := s.cloud(cfg)

	if cfg.TokenExpiresAt != nil && !s.now().Add(tokenRefreshSkew).Before(*cfg.TokenExpiresAt) {
		if err := s.refreshCloudClient(client, &cfg); err != nil {
			return zero, err
		}
	}

	result, requestErr := operation(client)
	if err := s.persistCloudCredentials(&cfg, client.Credentials()); err != nil {
		return zero, err
	}
	if requestErr == nil {
		return result, nil
	}
	if !isTokenAuthError(requestErr) {
		return zero, requestErr
	}
	if err := s.refreshCloudClient(client, &cfg); err != nil {
		return zero, err
	}

	result, requestErr = operation(client)
	if err := s.persistCloudCredentials(&cfg, client.Credentials()); err != nil {
		return zero, err
	}
	if requestErr != nil {
		return zero, requestErr
	}
	return result, nil
}

func (s *Service) refreshCloudClient(client cloudClient, cfg *model.GuangyaConfig) error {
	if strings.TrimSpace(cfg.RefreshToken) == "" {
		return errors.New("光鸭云盘访问令牌已过期且没有刷新令牌，请重新绑定")
	}
	if _, err := client.RefreshToken(cfg.RefreshToken); err != nil {
		return fmt.Errorf("刷新光鸭云盘访问令牌失败: %w", err)
	}
	if err := s.persistCloudCredentials(cfg, client.Credentials()); err != nil {
		return fmt.Errorf("保存光鸭云盘访问令牌失败: %w", err)
	}
	return nil
}

func (s *Service) persistCloudCredentials(cfg *model.GuangyaConfig, creds credentials) error {
	updates := make(map[string]any)
	if creds.AccessToken != "" && creds.AccessToken != cfg.AccessToken {
		updates["access_token"] = creds.AccessToken
		cfg.AccessToken = creds.AccessToken
	}
	if creds.RefreshToken != "" && creds.RefreshToken != cfg.RefreshToken {
		updates["refresh_token"] = creds.RefreshToken
		cfg.RefreshToken = creds.RefreshToken
	}
	if creds.DeviceID != "" && creds.DeviceID != cfg.DeviceID {
		updates["device_id"] = creds.DeviceID
		cfg.DeviceID = creds.DeviceID
	}
	if !creds.ExpiresAt.IsZero() && (cfg.TokenExpiresAt == nil || !creds.ExpiresAt.Equal(*cfg.TokenExpiresAt)) {
		expiresAt := creds.ExpiresAt
		updates["token_expires_at"] = expiresAt
		cfg.TokenExpiresAt = &expiresAt
	}
	if len(updates) == 0 {
		return nil
	}
	return s.db.Model(&model.GuangyaConfig{}).Where("id = ?", cfg.ID).Updates(updates).Error
}

func isTokenAuthError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "http 401") || strings.Contains(message, "status 401") {
		return true
	}
	if !strings.Contains(message, "http 400") && !strings.Contains(message, "status 400") {
		return false
	}
	for _, marker := range []string{
		"token", "access_token", "expired", "unauthorized", "unauthenticated",
		"invalid_grant", "authentication", "authorization", "令牌", "鉴权",
	} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func (s *Service) ResolveURL(url string) (CloudResolved, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return CloudResolved{}, errors.New("磁链不能为空")
	}
	result, err := cloudOperation(s, func(client cloudClient) (guangyaclient.CloudResolveURLResponse, error) {
		return client.CloudResolveURL(url)
	})
	if err != nil {
		return CloudResolved{}, err
	}
	info := result.Data.BTResInfo
	return CloudResolved{
		ResType:  result.Data.ResType,
		URL:      result.Data.URL,
		InfoHash: info.InfoHash,
		FileName: info.FileName,
		FileSize: info.FileSize,
		Subfiles: cloudFiles(info.Subfiles),
	}, nil
}

func (s *Service) Folders(parentID string, page, pageSize int) (CloudFolderList, error) {
	if page < 0 {
		page = 0
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 50
	}
	resType := 2
	result, err := cloudOperation(s, func(client cloudClient) (guangyaclient.FileListResponse, error) {
		return client.FSFiles(strings.TrimSpace(parentID), page, pageSize, 0, 0, nil, &resType, nil, false)
	})
	if err != nil {
		return CloudFolderList{}, err
	}
	folders := make([]CloudFolder, 0, len(result.Data.List))
	for _, item := range result.Data.List {
		folders = append(folders, CloudFolder{
			FileID: item.FileID, FileName: item.FileName, FileSize: item.FileSize,
			ResType: item.ResType, ParentID: item.ParentID,
		})
	}
	return CloudFolderList{List: folders, Total: result.Data.Total}, nil
}

func (s *Service) CreateFolder(name, parentID string) (CloudFolder, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CloudFolder{}, errors.New("文件夹名称不能为空")
	}
	result, err := cloudOperation(s, func(client cloudClient) (guangyaclient.FSCreateDirResponse, error) {
		return client.FSCreateDir(name, strings.TrimSpace(parentID), true)
	})
	if err != nil {
		return CloudFolder{}, err
	}
	return CloudFolder{
		FileID:   result.Data.FileID,
		FileName: result.Data.FileName,
		ResType:  result.Data.ResType,
		ParentID: result.Data.ParentID,
	}, nil
}

func (s *Service) CreateCloudTask(input CreateCloudTaskInput) (CloudTask, error) {
	url := strings.TrimSpace(input.URL)
	if url == "" {
		return CloudTask{}, errors.New("磁链不能为空")
	}
	newName := strings.TrimSpace(input.NewName)
	if newName == "" {
		return CloudTask{}, errors.New("名称不能为空")
	}
	if err := s.prepareSmartStrmTarget(&input); err != nil {
		return CloudTask{}, err
	}
	result, err := cloudOperation(s, func(client cloudClient) (guangyaclient.CloudCreateTaskResponse, error) {
		return client.CloudCreateTask(input.FileIndexes, url, strings.TrimSpace(input.ParentID), newName)
	})
	if err != nil {
		return CloudTask{}, err
	}
	if err := s.saveCloudAddJob(result.Data.TaskID, input, newName); err != nil {
		return CloudTask{}, fmt.Errorf("保存云添加后处理任务失败: %w", err)
	}
	return CloudTask{TaskID: result.Data.TaskID, URL: result.Data.URL}, nil
}

func (s *Service) prepareSmartStrmTarget(input *CreateCloudTaskInput) error {
	var mapping model.StorageMapping
	var err error
	if input.StorageMappingID > 0 {
		err = s.db.First(&mapping, input.StorageMappingID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("所选存储配置不存在")
		}
		if err != nil {
			return fmt.Errorf("读取存储配置失败: %w", err)
		}
		if strings.TrimSpace(mapping.GuangyaFolderID) != strings.TrimSpace(input.ParentID) {
			return errors.New("所选存储配置与保存目录不一致")
		}
	} else {
		err = s.db.Where("guangya_folder_id = ?", strings.TrimSpace(input.ParentID)).
			Order("id asc").
			First(&mapping).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("读取存储配置失败: %w", err)
		}
	}
	input.SmartStrmTaskName = strings.TrimSpace(mapping.SmartStrmTaskName)
	input.SmartStrmStorageRoot = strings.TrimSpace(mapping.GuangyaFolderPath)
	input.StrmSavePath = strings.TrimSpace(mapping.StrmSavePath)
	return nil
}

func cloudFiles(files []guangyaclient.CloudSubfile) []CloudFile {
	result := make([]CloudFile, 0, len(files))
	for _, file := range files {
		result = append(result, CloudFile{
			FileName: file.FileName, FileSize: file.FileSize, FileIndex: file.FileIndex,
			IsDir: file.IsDir, Subfiles: cloudFiles(file.Subfiles),
		})
	}
	return result
}

func (s *Service) Config() (ConfigView, error) {
	var cfg model.GuangyaConfig
	err := s.db.Order("id desc").First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ConfigView{}, nil
	}
	if err != nil {
		return ConfigView{}, err
	}
	return configView(cfg), nil
}

func (s *Service) Unbind() error {
	s.mu.Lock()
	s.sessions = make(map[string]*loginSession)
	s.mu.Unlock()
	return s.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.GuangyaConfig{}).Error
}

func (s *Service) StartQR() (QRStartResult, error) {
	s.cleanupExpired()
	client := s.newClient()
	result, err := client.DeviceCode(defaultQRScope)
	if err != nil {
		return QRStartResult{}, err
	}
	if result.DeviceCode == "" {
		return QRStartResult{}, errors.New("光鸭云盘未返回设备授权码")
	}

	interval := result.Interval
	if interval < 2 {
		interval = 2
	}
	expiresIn := result.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = int(loginSessionTTL.Seconds())
	}
	expiresAt := s.now().Add(time.Duration(expiresIn) * time.Second)
	sessionID, err := newSessionID()
	if err != nil {
		return QRStartResult{}, err
	}

	s.mu.Lock()
	s.sessions[sessionID] = &loginSession{
		client:     client,
		expiresAt:  expiresAt,
		deviceCode: result.DeviceCode,
	}
	s.mu.Unlock()

	return QRStartResult{
		SessionID:               sessionID,
		UserCode:                result.UserCode,
		VerificationURL:         result.VerificationURL,
		VerificationURIComplete: result.VerificationURIComplete,
		Interval:                interval,
		ExpiresAt:               expiresAt,
	}, nil
}

func (s *Service) PollQR(sessionID string) (QRPollResult, error) {
	session, err := s.session(sessionID)
	if err != nil {
		return QRPollResult{}, err
	}
	session.mu.Lock()
	defer session.mu.Unlock()

	if !s.now().Before(session.expiresAt) {
		s.deleteSession(sessionID)
		return QRPollResult{State: "expired"}, nil
	}

	token, err := session.client.DeviceToken(session.deviceCode)
	if err != nil {
		switch classifyDeviceError(err) {
		case "pending":
			return QRPollResult{State: "pending"}, nil
		case "expired":
			s.deleteSession(sessionID)
			return QRPollResult{State: "expired"}, nil
		case "denied":
			s.deleteSession(sessionID)
			return QRPollResult{State: "denied"}, nil
		default:
			return QRPollResult{}, err
		}
	}

	cfg, err := s.saveBinding(session.client, token, "qr", "")
	if err != nil {
		return QRPollResult{}, err
	}
	s.deleteSession(sessionID)
	view := configView(cfg)
	return QRPollResult{State: "bound", Config: &view}, nil
}

func (s *Service) SendSMS(phone, captchaToken, sessionID string) (SMSSendResult, error) {
	s.cleanupExpired()
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return SMSSendResult{}, errors.New("请输入手机号")
	}

	var session *loginSession
	var err error
	sessionID = strings.TrimSpace(sessionID)
	if sessionID != "" {
		session, err = s.session(sessionID)
		if err != nil {
			return SMSSendResult{}, err
		}
		if session.phone != phone {
			return SMSSendResult{}, errors.New("手机号与登录会话不匹配，请重新发起绑定")
		}
	} else {
		sessionID, err = newSessionID()
		if err != nil {
			return SMSSendResult{}, err
		}
		session = &loginSession{
			client:    s.newClient(),
			expiresAt: s.now().Add(loginSessionTTL),
			phone:     phone,
		}
		s.mu.Lock()
		s.sessions[sessionID] = session
		s.mu.Unlock()
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	client := session.client
	initResult, err := client.LoginSMSInit(phone, strings.TrimSpace(captchaToken))
	if err != nil {
		return SMSSendResult{}, err
	}
	if initResult.CaptchaToken == "" {
		if initResult.URL != "" {
			return SMSSendResult{
				SessionID:       sessionID,
				CaptchaRequired: true,
				CaptchaURL:      initResult.URL,
			}, nil
		}
		return SMSSendResult{}, errors.New("光鸭云盘未返回短信验证令牌")
	}

	sendResult, err := client.LoginSMSSend(phone, initResult.CaptchaToken, defaultSMSTarget)
	if err != nil {
		return SMSSendResult{}, err
	}
	if sendResult.VerificationID == "" {
		return SMSSendResult{}, errors.New("光鸭云盘未返回短信验证 ID")
	}

	session.expiresAt = s.now().Add(loginSessionTTL)
	session.captchaToken = initResult.CaptchaToken
	session.verificationID = sendResult.VerificationID

	return SMSSendResult{SessionID: sessionID}, nil
}

func (s *Service) VerifySMS(sessionID, code string) (ConfigView, error) {
	session, err := s.session(sessionID)
	if err != nil {
		return ConfigView{}, err
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return ConfigView{}, errors.New("请输入短信验证码")
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	if !s.now().Before(session.expiresAt) {
		s.deleteSession(sessionID)
		return ConfigView{}, ErrSessionExpired
	}
	if session.verificationID == "" {
		return ConfigView{}, errors.New("短信登录会话无效，请重新发送验证码")
	}

	verifyResult, err := session.client.LoginSMSVerify(session.verificationID, code)
	if err != nil {
		return ConfigView{}, err
	}
	if verifyResult.VerificationToken == "" {
		return ConfigView{}, errors.New("光鸭云盘未返回验证码确认令牌")
	}
	token, err := session.client.LoginSMSSignin(code, verifyResult.VerificationToken, session.phone, session.captchaToken)
	if err != nil {
		return ConfigView{}, err
	}

	cfg, err := s.saveBinding(session.client, token, "sms", session.phone)
	if err != nil {
		return ConfigView{}, err
	}
	s.deleteSession(sessionID)
	return configView(cfg), nil
}

func (s *Service) saveBinding(client authClient, token guangyaclient.TokenResponse, method, fallbackPhone string) (model.GuangyaConfig, error) {
	creds := client.Credentials()
	if creds.AccessToken == "" {
		creds.AccessToken = token.AccessToken
	}
	if creds.RefreshToken == "" {
		creds.RefreshToken = token.RefreshToken
	}
	if creds.AccessToken == "" {
		return model.GuangyaConfig{}, errors.New("光鸭云盘未返回访问令牌")
	}

	user, userErr := client.UserInfo()
	phone := user.PhoneNumber
	if phone == "" {
		phone = fallbackPhone
	}
	subject := user.Subject
	if subject == "" {
		subject = token.Subject
	}
	var expiresAt *time.Time
	if !creds.ExpiresAt.IsZero() {
		value := creds.ExpiresAt
		expiresAt = &value
	} else if token.ExpiresIn > 0 {
		value := s.now().Add(time.Duration(token.ExpiresIn) * time.Second)
		expiresAt = &value
	}

	cfg := model.GuangyaConfig{
		AccessToken:    creds.AccessToken,
		RefreshToken:   creds.RefreshToken,
		DeviceID:       creds.DeviceID,
		Subject:        subject,
		Name:           user.Name,
		PhoneNumber:    phone,
		LoginMethod:    method,
		TokenExpiresAt: expiresAt,
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.GuangyaConfig{}).Error; err != nil {
			return err
		}
		return tx.Create(&cfg).Error
	})
	if err != nil {
		return model.GuangyaConfig{}, err
	}
	// UserInfo may be unavailable immediately after login; valid tokens are still persisted.
	_ = userErr
	return cfg, nil
}

func (s *Service) session(sessionID string) (*loginSession, error) {
	sessionID = strings.TrimSpace(sessionID)
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	if !s.now().Before(session.expiresAt) {
		delete(s.sessions, sessionID)
		return nil, ErrSessionExpired
	}
	return session, nil
}

func (s *Service) cleanupExpired() {
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, session := range s.sessions {
		if !now.Before(session.expiresAt) {
			delete(s.sessions, id)
		}
	}
}

func (s *Service) deleteSession(sessionID string) {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
}

func configView(cfg model.GuangyaConfig) ConfigView {
	updatedAt := cfg.UpdatedAt
	return ConfigView{
		Bound:          cfg.AccessToken != "",
		Subject:        cfg.Subject,
		Name:           cfg.Name,
		PhoneNumber:    maskPhone(cfg.PhoneNumber),
		LoginMethod:    cfg.LoginMethod,
		TokenExpiresAt: cfg.TokenExpiresAt,
		UpdatedAt:      &updatedAt,
	}
}

func maskPhone(phone string) string {
	value := strings.TrimSpace(phone)
	digitIndexes := make([]int, 0, len(value))
	for index, char := range value {
		if char >= '0' && char <= '9' {
			digitIndexes = append(digitIndexes, index)
		}
	}
	if len(digitIndexes) < 7 {
		if value == "" {
			return ""
		}
		return "***"
	}
	masked := []byte(value)
	for _, index := range digitIndexes[len(digitIndexes)-8 : len(digitIndexes)-4] {
		masked[index] = '*'
	}
	return string(masked)
}

func classifyDeviceError(err error) string {
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "authorization_pending"), strings.Contains(message, "slow_down"):
		return "pending"
	case strings.Contains(message, "expired_token"):
		return "expired"
	case strings.Contains(message, "access_denied"):
		return "denied"
	default:
		return ""
	}
}

func newSessionID() (string, error) {
	var value [24]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("生成登录会话失败: %w", err)
	}
	return hex.EncodeToString(value[:]), nil
}
