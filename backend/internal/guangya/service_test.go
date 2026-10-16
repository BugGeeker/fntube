package guangya

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	guangyaclient "github.com/BugGeeker/guangyaclient-go"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"fntube/internal/model"
)

type fakeClient struct {
	deviceCodeResult guangyaclient.DeviceCodeResponse
	deviceToken      guangyaclient.TokenResponse
	deviceTokenErr   error
	captcha          guangyaclient.CaptchaInitResponse
	smsSend          guangyaclient.SMSVerificationResponse
	smsVerify        guangyaclient.SMSVerifyResponse
	smsToken         guangyaclient.TokenResponse
	user             guangyaclient.UserInfoResponse
	creds            credentials
	captchaResults   []guangyaclient.CaptchaInitResponse
	captchaCalls     int
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type fakeCloudClient struct {
	resolveResult   guangyaclient.CloudResolveURLResponse
	resolveErrs     []error
	resolveCalls    int
	resolveURL      string
	foldersResult   guangyaclient.FileListResponse
	foldersErr      error
	folderParent    any
	folderPage      int
	folderSize      int
	createDirResult guangyaclient.FSCreateDirResponse
	createDirErr    error
	createDirName   string
	createDirParent any
	createDirFail   bool
	taskResult      guangyaclient.CloudCreateTaskResponse
	taskErr         error
	taskIndexes     []int
	taskURL         string
	taskParent      any
	taskName        string
	taskListResult  guangyaclient.CloudTaskListResponse
	taskListErr     error
	taskListIDs     []string
	filesByParent   map[string]guangyaclient.FileListResponse
	moveResult      guangyaclient.FileTaskResponse
	moveErr         error
	moveFileIDs     [][]any
	moveParents     []any
	renameErr       error
	renameFileIDs   []string
	renameNames     []string
	fileTaskStatus  guangyaclient.TaskStatusResponse
	fileTaskErr     error
	refreshResult   guangyaclient.TokenResponse
	refreshErr      error
	refreshCalls    int
	refreshValue    string
	refreshExpiry   time.Time
	creds           credentials
}

func (f *fakeCloudClient) CloudResolveURL(url string) (guangyaclient.CloudResolveURLResponse, error) {
	f.resolveURL = url
	index := f.resolveCalls
	f.resolveCalls++
	if index < len(f.resolveErrs) {
		return f.resolveResult, f.resolveErrs[index]
	}
	return f.resolveResult, nil
}

func (f *fakeCloudClient) FSFiles(parentID any, page, pageSize, _, _ int, _ []int, _, _ *int, _ bool) (guangyaclient.FileListResponse, error) {
	f.folderParent = parentID
	f.folderPage = page
	f.folderSize = pageSize
	if f.filesByParent != nil {
		if result, ok := f.filesByParent[fmt.Sprint(parentID)]; ok {
			return result, f.foldersErr
		}
	}
	return f.foldersResult, f.foldersErr
}

func (f *fakeCloudClient) FSCreateDir(dirName string, parentID any, failIfNameExist bool) (guangyaclient.FSCreateDirResponse, error) {
	f.createDirName = dirName
	f.createDirParent = parentID
	f.createDirFail = failIfNameExist
	return f.createDirResult, f.createDirErr
}

func (f *fakeCloudClient) CloudCreateTask(fileIndexes []int, url string, parentID any, newName string) (guangyaclient.CloudCreateTaskResponse, error) {
	f.taskIndexes = fileIndexes
	f.taskURL = url
	f.taskParent = parentID
	f.taskName = newName
	return f.taskResult, f.taskErr
}

func (f *fakeCloudClient) CloudTaskList(_, _ int, _ []int, taskIDs []string) (guangyaclient.CloudTaskListResponse, error) {
	f.taskListIDs = append([]string(nil), taskIDs...)
	return f.taskListResult, f.taskListErr
}

func (f *fakeCloudClient) FSMove(fileIDs []any, parentID any) (guangyaclient.FileTaskResponse, error) {
	f.moveFileIDs = append(f.moveFileIDs, append([]any(nil), fileIDs...))
	f.moveParents = append(f.moveParents, parentID)
	return f.moveResult, f.moveErr
}

func (f *fakeCloudClient) FSRename(fileID, newName string) (guangyaclient.EmptyResponse, error) {
	f.renameFileIDs = append(f.renameFileIDs, fileID)
	f.renameNames = append(f.renameNames, newName)
	return guangyaclient.EmptyResponse{}, f.renameErr
}

func (f *fakeCloudClient) GetTaskStatus(string) (guangyaclient.TaskStatusResponse, error) {
	return f.fileTaskStatus, f.fileTaskErr
}

func (f *fakeCloudClient) RefreshToken(refresh string) (guangyaclient.TokenResponse, error) {
	f.refreshCalls++
	f.refreshValue = refresh
	if f.refreshErr != nil {
		return f.refreshResult, f.refreshErr
	}
	if f.refreshResult.AccessToken != "" {
		f.creds.AccessToken = f.refreshResult.AccessToken
	}
	if f.refreshResult.RefreshToken != "" {
		f.creds.RefreshToken = f.refreshResult.RefreshToken
	}
	if f.refreshResult.ExpiresIn > 0 {
		if !f.refreshExpiry.IsZero() {
			f.creds.ExpiresAt = f.refreshExpiry
		} else {
			f.creds.ExpiresAt = time.Now().Add(time.Duration(f.refreshResult.ExpiresIn) * time.Second)
		}
	}
	return f.refreshResult, nil
}

func (f *fakeCloudClient) Credentials() credentials {
	return f.creds
}

func (f *fakeClient) DeviceCode(string) (guangyaclient.DeviceCodeResponse, error) {
	return f.deviceCodeResult, nil
}
func (f *fakeClient) DeviceToken(string) (guangyaclient.TokenResponse, error) {
	return f.deviceToken, f.deviceTokenErr
}
func (f *fakeClient) LoginSMSInit(string, string) (guangyaclient.CaptchaInitResponse, error) {
	if f.captchaCalls < len(f.captchaResults) {
		result := f.captchaResults[f.captchaCalls]
		f.captchaCalls++
		return result, nil
	}
	f.captchaCalls++
	return f.captcha, nil
}
func (f *fakeClient) LoginSMSSend(string, string, string) (guangyaclient.SMSVerificationResponse, error) {
	return f.smsSend, nil
}
func (f *fakeClient) LoginSMSVerify(string, string) (guangyaclient.SMSVerifyResponse, error) {
	return f.smsVerify, nil
}
func (f *fakeClient) LoginSMSSignin(string, string, string, string) (guangyaclient.TokenResponse, error) {
	return f.smsToken, nil
}
func (f *fakeClient) UserInfo() (guangyaclient.UserInfoResponse, error) {
	return f.user, nil
}
func (f *fakeClient) Credentials() credentials {
	return f.creds
}

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.GuangyaConfig{}, &model.GuangyaCloudAddJob{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get database connection: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	return db
}

func TestConfigMasksCredentials(t *testing.T) {
	db := testDB(t)
	cfg := model.GuangyaConfig{
		AccessToken:  "access-secret",
		RefreshToken: "refresh-secret",
		DeviceID:     "device-secret",
		PhoneNumber:  "+86 13800138000",
		LoginMethod:  "sms",
	}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}

	service := newService(db, func() authClient { return &fakeClient{} })
	view, err := service.Config()
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !view.Bound || view.PhoneNumber != "+86 138****8000" {
		t.Fatalf("unexpected config view: %#v", view)
	}
	if strings.Contains(view.PhoneNumber, "00138000") {
		t.Fatalf("phone number was not masked: %s", view.PhoneNumber)
	}
}

func TestPollQRReturnsPendingWithoutRemovingSession(t *testing.T) {
	db := testDB(t)
	client := &fakeClient{
		deviceCodeResult: guangyaclient.DeviceCodeResponse{
			DeviceCode: "device-code",
			UserCode:   "user-code",
			ExpiresIn:  300,
			Interval:   1,
		},
		deviceTokenErr: errors.New(`guangya: HTTP 400 Bad Request: {"error":"authorization_pending"}`),
	}
	service := newService(db, func() authClient { return client })
	started, err := service.StartQR()
	if err != nil {
		t.Fatalf("start qr: %v", err)
	}

	result, err := service.PollQR(started.SessionID)
	if err != nil {
		t.Fatalf("poll qr: %v", err)
	}
	if result.State != "pending" {
		t.Fatalf("unexpected state: %s", result.State)
	}
	if _, err := service.session(started.SessionID); err != nil {
		t.Fatalf("pending session was removed: %v", err)
	}
}

func TestVerifySMSSavesTokensAndAccount(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	client := &fakeClient{
		captcha:   guangyaclient.CaptchaInitResponse{CaptchaToken: "captcha"},
		smsSend:   guangyaclient.SMSVerificationResponse{VerificationID: "verification"},
		smsVerify: guangyaclient.SMSVerifyResponse{VerificationToken: "verified"},
		smsToken: guangyaclient.TokenResponse{
			AccessToken:  "access",
			RefreshToken: "refresh",
			ExpiresIn:    3600,
			Subject:      "subject",
		},
		user: guangyaclient.UserInfoResponse{
			Subject:     "subject",
			Name:        "Test User",
			PhoneNumber: "+86 13800138000",
		},
		creds: credentials{
			AccessToken:  "access",
			RefreshToken: "refresh",
			DeviceID:     "device",
			ExpiresAt:    now.Add(time.Hour),
		},
	}
	service := newService(db, func() authClient { return client })
	service.now = func() time.Time { return now }

	sent, err := service.SendSMS("+86 13800138000", "", "")
	if err != nil {
		t.Fatalf("send sms: %v", err)
	}
	view, err := service.VerifySMS(sent.SessionID, "123456")
	if err != nil {
		t.Fatalf("verify sms: %v", err)
	}
	if !view.Bound || view.Name != "Test User" || view.PhoneNumber != "+86 138****8000" {
		t.Fatalf("unexpected config view: %#v", view)
	}

	var saved model.GuangyaConfig
	if err := db.First(&saved).Error; err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if saved.AccessToken != "access" || saved.RefreshToken != "refresh" || saved.DeviceID != "device" {
		t.Fatalf("credentials were not persisted: %#v", saved)
	}
	if _, err := service.session(sent.SessionID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("successful session was not removed: %v", err)
	}
}

func TestSendSMSReusesSessionAfterCaptcha(t *testing.T) {
	db := testDB(t)
	client := &fakeClient{
		captchaResults: []guangyaclient.CaptchaInitResponse{
			{URL: "https://account.example/captcha"},
			{CaptchaToken: "captcha"},
		},
		smsSend: guangyaclient.SMSVerificationResponse{VerificationID: "verification"},
	}
	factoryCalls := 0
	service := newService(db, func() authClient {
		factoryCalls++
		return client
	})

	pending, err := service.SendSMS("+86 13800138000", "", "")
	if err != nil {
		t.Fatalf("start captcha: %v", err)
	}
	if !pending.CaptchaRequired || pending.SessionID == "" {
		t.Fatalf("unexpected captcha result: %#v", pending)
	}

	sent, err := service.SendSMS("+86 13800138000", "", pending.SessionID)
	if err != nil {
		t.Fatalf("send sms after captcha: %v", err)
	}
	if sent.SessionID != pending.SessionID || sent.CaptchaRequired {
		t.Fatalf("unexpected send result: %#v", sent)
	}
	if factoryCalls != 1 {
		t.Fatalf("captcha retry created %d clients, want 1", factoryCalls)
	}
}

func TestVerifySMSRejectsExpiredSession(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	service := newService(db, func() authClient { return &fakeClient{} })
	service.now = func() time.Time { return now }
	service.sessions["expired"] = &loginSession{
		client:    &fakeClient{},
		expiresAt: now.Add(-time.Second),
	}

	_, err := service.VerifySMS("expired", "123456")
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected expired session error, got %v", err)
	}
}

func TestCloudOperationsRequireBinding(t *testing.T) {
	service := newService(testDB(t), func() authClient { return &fakeClient{} })
	if _, err := service.ResolveURL("magnet:?xt=urn:btih:test"); !errors.Is(err, ErrNotBound) {
		t.Fatalf("expected unbound error, got %v", err)
	}
}

func TestCloudOperationsPassThroughSDKData(t *testing.T) {
	db := testDB(t)
	cfg := model.GuangyaConfig{AccessToken: "access", RefreshToken: "refresh", DeviceID: "device"}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}
	index := 7
	client := &fakeCloudClient{
		resolveResult: guangyaclient.CloudResolveURLResponse{
			Data: guangyaclient.CloudResolveURLData{
				ResType: 1,
				URL:     "magnet:?xt=urn:btih:resolved",
				BTResInfo: guangyaclient.BTResourceInfo{
					InfoHash: "hash",
					FileName: "movie",
					FileSize: 100,
					Subfiles: []guangyaclient.CloudSubfile{{FileName: "movie.mkv", FileIndex: &index}},
				},
			},
		},
		foldersResult: guangyaclient.FileListResponse{
			Data: guangyaclient.FileListData{
				Total: 1,
				List:  []guangyaclient.FileItem{{FileID: "folder", FileName: "Movies", ResType: 2}},
			},
		},
		createDirResult: guangyaclient.FSCreateDirResponse{
			Data: guangyaclient.FSCreateDirData{
				FileID: "new-folder", FileName: "新目录", ParentID: "parent", ResType: 2,
			},
		},
		taskResult: guangyaclient.CloudCreateTaskResponse{
			Data: guangyaclient.CloudCreateTaskData{TaskID: "task", URL: "magnet:?xt=urn:btih:resolved"},
		},
	}
	service := newService(db, func() authClient { return &fakeClient{} })
	service.newCloud = func(credentials) cloudClient { return client }

	resolved, err := service.ResolveURL("  magnet:?xt=urn:btih:test  ")
	if err != nil {
		t.Fatalf("resolve url: %v", err)
	}
	if client.resolveURL != "magnet:?xt=urn:btih:test" || resolved.InfoHash != "hash" ||
		len(resolved.Subfiles) != 1 || resolved.Subfiles[0].FileIndex == nil {
		t.Fatalf("unexpected resolve result: %#v", resolved)
	}

	folders, err := service.Folders("parent", 2, 25)
	if err != nil {
		t.Fatalf("list folders: %v", err)
	}
	if client.folderParent != "parent" || client.folderPage != 2 || client.folderSize != 25 ||
		len(folders.List) != 1 || folders.List[0].FileID != "folder" {
		t.Fatalf("unexpected folders: %#v", folders)
	}

	created, err := service.CreateFolder("  新目录  ", " parent ")
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if created.FileID != "new-folder" || created.FileName != "新目录" ||
		client.createDirName != "新目录" || client.createDirParent != "parent" || !client.createDirFail {
		t.Fatalf("unexpected created folder: %#v %#v", created, client)
	}

	task, err := service.CreateCloudTask(CreateCloudTaskInput{
		FileIndexes: []int{7},
		URL:         " magnet:?xt=urn:btih:resolved ",
		ParentID:    "parent",
		NewName:     "movie",
	})
	if err != nil {
		t.Fatalf("create cloud task: %v", err)
	}
	if task.TaskID != "task" || client.taskURL != "magnet:?xt=urn:btih:resolved" ||
		client.taskParent != "parent" || client.taskName != "movie" || len(client.taskIndexes) != 1 ||
		client.taskIndexes[0] != 7 {
		t.Fatalf("unexpected task: %#v %#v", task, client)
	}
}

func TestCloudOperationRefreshesExpiredTokenAndPersistsCredentials(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	expiredAt := now.Add(-time.Minute)
	cfg := model.GuangyaConfig{
		AccessToken:    "expired-access",
		RefreshToken:   "old-refresh",
		DeviceID:       "device",
		TokenExpiresAt: &expiredAt,
	}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}

	client := &fakeCloudClient{
		foldersResult: guangyaclient.FileListResponse{
			Data: guangyaclient.FileListData{
				List: []guangyaclient.FileItem{{FileID: "folder", FileName: "Movies", ResType: 2}},
			},
		},
		refreshResult: guangyaclient.TokenResponse{
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
			ExpiresIn:    3600,
		},
		refreshExpiry: now.Add(time.Hour),
		creds: credentials{
			AccessToken:  "expired-access",
			RefreshToken: "old-refresh",
			DeviceID:     "device",
			ExpiresAt:    expiredAt,
		},
	}
	service := newService(db, func() authClient { return &fakeClient{} })
	service.now = func() time.Time { return now }
	var factoryCreds credentials
	service.newCloud = func(creds credentials) cloudClient {
		factoryCreds = creds
		return client
	}

	folders, err := service.Folders("", 0, 50)
	if err != nil {
		t.Fatalf("list folders: %v", err)
	}
	if len(folders.List) != 1 || client.refreshCalls != 1 || client.refreshValue != "old-refresh" {
		t.Fatalf("unexpected refresh result: folders=%#v client=%#v", folders, client)
	}
	if !factoryCreds.ExpiresAt.Equal(expiredAt) {
		t.Fatalf("database expiry was not passed to SDK client: %v", factoryCreds.ExpiresAt)
	}

	var saved model.GuangyaConfig
	if err := db.First(&saved, cfg.ID).Error; err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if saved.AccessToken != "new-access" || saved.RefreshToken != "new-refresh" ||
		saved.TokenExpiresAt == nil || !saved.TokenExpiresAt.After(now) {
		t.Fatalf("refreshed credentials were not persisted: %#v", saved)
	}
}

func TestCloudOperationRefreshesAndRetriesTokenErrorOnce(t *testing.T) {
	db := testDB(t)
	now := time.Now()
	expiresAt := now.Add(time.Hour)
	cfg := model.GuangyaConfig{
		AccessToken:    "access",
		RefreshToken:   "refresh",
		DeviceID:       "device",
		TokenExpiresAt: &expiresAt,
	}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}

	client := &fakeCloudClient{
		resolveResult: guangyaclient.CloudResolveURLResponse{
			Data: guangyaclient.CloudResolveURLData{URL: "magnet:?xt=urn:btih:resolved"},
		},
		resolveErrs: []error{
			errors.New(`guangya: HTTP 400 Bad Request: {"error":"access_token_expired"}`),
			nil,
		},
		refreshResult: guangyaclient.TokenResponse{
			AccessToken: "new-access",
			ExpiresIn:   3600,
		},
		creds: credentials{
			AccessToken:  "access",
			RefreshToken: "refresh",
			DeviceID:     "device",
			ExpiresAt:    expiresAt,
		},
	}
	service := newService(db, func() authClient { return &fakeClient{} })
	service.newCloud = func(credentials) cloudClient { return client }

	result, err := service.ResolveURL("magnet:?xt=urn:btih:test")
	if err != nil {
		t.Fatalf("resolve url: %v", err)
	}
	if result.URL != "magnet:?xt=urn:btih:resolved" || client.resolveCalls != 2 || client.refreshCalls != 1 {
		t.Fatalf("request was not refreshed and retried once: result=%#v client=%#v", result, client)
	}
}

func TestCloudOperationPersistsCredentialsRefreshedBySDK(t *testing.T) {
	db := testDB(t)
	expiresAt := time.Now().Add(time.Hour)
	cfg := model.GuangyaConfig{
		AccessToken:    "old-access",
		RefreshToken:   "old-refresh",
		DeviceID:       "device",
		TokenExpiresAt: &expiresAt,
	}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}

	refreshedAt := expiresAt.Add(time.Hour)
	client := &fakeCloudClient{
		creds: credentials{
			AccessToken:  "sdk-access",
			RefreshToken: "sdk-refresh",
			DeviceID:     "device",
			ExpiresAt:    refreshedAt,
		},
	}
	service := newService(db, func() authClient { return &fakeClient{} })
	service.newCloud = func(credentials) cloudClient { return client }

	if _, err := service.Folders("", 0, 50); err != nil {
		t.Fatalf("list folders: %v", err)
	}

	var saved model.GuangyaConfig
	if err := db.First(&saved, cfg.ID).Error; err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if saved.AccessToken != "sdk-access" || saved.RefreshToken != "sdk-refresh" ||
		saved.TokenExpiresAt == nil || !saved.TokenExpiresAt.Equal(refreshedAt) {
		t.Fatalf("SDK credentials were not persisted: %#v", saved)
	}
}

func TestCloudOperationExpiredWithoutRefreshTokenRequiresRebind(t *testing.T) {
	db := testDB(t)
	now := time.Now()
	expiredAt := now.Add(-time.Minute)
	cfg := model.GuangyaConfig{
		AccessToken:    "expired-access",
		DeviceID:       "device",
		TokenExpiresAt: &expiredAt,
	}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}

	client := &fakeCloudClient{}
	service := newService(db, func() authClient { return &fakeClient{} })
	service.now = func() time.Time { return now }
	service.newCloud = func(credentials) cloudClient { return client }

	_, err := service.Folders("", 0, 50)
	if err == nil || !strings.Contains(err.Error(), "没有刷新令牌") {
		t.Fatalf("expected missing refresh token error, got %v", err)
	}
	if client.refreshCalls != 0 {
		t.Fatalf("refresh was attempted without a refresh token")
	}
}

func TestCloudOperationDoesNotRetryUnrelatedBadRequest(t *testing.T) {
	db := testDB(t)
	cfg := model.GuangyaConfig{AccessToken: "access", RefreshToken: "refresh", DeviceID: "device"}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}

	client := &fakeCloudClient{
		resolveErrs: []error{errors.New(`guangya: HTTP 400 Bad Request: {"error":"invalid_url"}`)},
		creds: credentials{
			AccessToken:  "access",
			RefreshToken: "refresh",
			DeviceID:     "device",
		},
	}
	service := newService(db, func() authClient { return &fakeClient{} })
	service.newCloud = func(credentials) cloudClient { return client }

	_, err := service.ResolveURL("not-a-url")
	if err == nil {
		t.Fatal("expected bad request error")
	}
	if client.resolveCalls != 1 || client.refreshCalls != 0 {
		t.Fatalf("unrelated bad request was retried: %#v", client)
	}
}

func TestSDKClientRefreshClearsExpiredDeadline(t *testing.T) {
	requests := 0
	client := guangyaclient.NewClient("expired-access", "refresh", "device")
	client.TokenExpiresAt = time.Now().Add(-time.Minute)
	client.HTTP = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(
					`{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`,
				)),
				Request: req,
			}, nil
		}),
	}

	result, err := (&sdkClient{Client: client}).RefreshToken("")
	if err != nil {
		t.Fatalf("refresh token: %v", err)
	}
	if requests != 1 || result.AccessToken != "new-access" || client.Token != "new-access" ||
		client.RefreshTokenValue != "new-refresh" || !client.TokenExpiresAt.After(time.Now()) {
		t.Fatalf("unexpected refresh result: requests=%d result=%#v client=%#v", requests, result, client)
	}
}
