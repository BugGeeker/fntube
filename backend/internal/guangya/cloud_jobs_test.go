package guangya

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	guangyaclient "github.com/BugGeeker/guangyaclient-go"

	"fntube/internal/model"
)

func newCloudJobTestService(t *testing.T, client *fakeCloudClient) *Service {
	t.Helper()
	db := testDB(t)
	if err := db.Create(&model.GuangyaConfig{
		AccessToken: "access", RefreshToken: "refresh", DeviceID: "device",
	}).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}
	service := newService(db, func() authClient { return &fakeClient{} })
	service.newCloud = func(credentials) cloudClient { return client }
	return service
}

func cloudJob(t *testing.T, directory bool, name string, selected []SelectedCloudFile) model.GuangyaCloudAddJob {
	t.Helper()
	data, err := json.Marshal(selected)
	if err != nil {
		t.Fatalf("marshal selected files: %v", err)
	}
	return model.GuangyaCloudAddJob{
		TaskID:              "cloud-task",
		ParentID:            "destination",
		RequestedName:       name,
		ResolvedIsDirectory: directory,
		SelectedFiles:       string(data),
		Status:              cloudJobPending,
	}
}

func completedCloudTask(name string, directory bool) guangyaclient.CloudTaskListResponse {
	return guangyaclient.CloudTaskListResponse{
		Data: guangyaclient.CloudTaskListData{
			Total: 1,
			List: []guangyaclient.CloudTask{{
				TaskID: "cloud-task", Status: 2, FileName: name, IsDir: directory,
			}},
		},
	}
}

func fileList(items ...guangyaclient.FileItem) guangyaclient.FileListResponse {
	return guangyaclient.FileListResponse{
		Data: guangyaclient.FileListData{List: items, Total: len(items)},
	}
}

func TestProcessSingleFileCreatesFolderAndMovesFile(t *testing.T) {
	client := &fakeCloudClient{
		taskListResult: completedCloudTask("RKI-404-C.mp4", false),
		createDirResults: []guangyaclient.FSCreateDirResponse{
			{Data: guangyaclient.FSCreateDirData{
				FileID: "collection-folder", FileName: "RKI", ParentID: "destination", ResType: 2,
			}},
			{Data: guangyaclient.FSCreateDirData{
				FileID: "target-folder", FileName: "RKI-404-C", ParentID: "collection-folder", ResType: 2,
			}},
		},
		filesByParent: map[string]guangyaclient.FileListResponse{
			"destination": fileList(guangyaclient.FileItem{
				FileID: "movie", FileName: "RKI-404-C.mp4", ParentID: "destination", ResType: 1,
			}),
			"collection-folder": fileList(),
			"target-folder":     fileList(),
		},
		moveResult: guangyaclient.FileTaskResponse{
			Data: guangyaclient.FileTaskData{TaskID: "move-task"},
		},
		fileTaskStatus: guangyaclient.TaskStatusResponse{
			Data: guangyaclient.TaskStatusData{Status: 2, Progress: 100},
		},
	}
	service := newCloudJobTestService(t, client)
	job := cloudJob(t, false, "RKI-404-C.mp4", nil)

	status, err := service.processCloudAddJob(&job)
	if err != nil {
		t.Fatalf("process job: %v", err)
	}
	if status != cloudJobCompleted ||
		len(client.createDirNames) != 2 ||
		client.createDirNames[0] != "RKI" ||
		client.createDirNames[1] != "RKI-404-C" {
		t.Fatalf("unexpected status or folders: status=%s folders=%#v", status, client.createDirNames)
	}
	if len(client.createDirParents) != 2 ||
		client.createDirParents[0] != "destination" ||
		client.createDirParents[1] != "collection-folder" {
		t.Fatalf("unexpected folder parents: %#v", client.createDirParents)
	}
	if len(client.taskListIDs) != 1 || client.taskListIDs[0] != "cloud-task" {
		t.Fatalf("cloud task was not queried by task ID: %#v", client.taskListIDs)
	}
	if len(client.moveFileIDs) != 1 || client.moveFileIDs[0][0] != "movie" ||
		client.moveParents[0] != "target-folder" {
		t.Fatalf("unexpected move calls: ids=%#v parents=%#v", client.moveFileIDs, client.moveParents)
	}
}

func TestProcessSingleFileReusesConfiguredCollectionFolder(t *testing.T) {
	client := &fakeCloudClient{
		taskListResult: completedCloudTask("RKI-404-C.mp4", false),
		filesByParent: map[string]guangyaclient.FileListResponse{
			"destination": fileList(guangyaclient.FileItem{
				FileID: "movie", FileName: "RKI-404-C.mp4", ParentID: "destination", ResType: 1,
			}),
			"target-folder": fileList(),
		},
		createDirResult: guangyaclient.FSCreateDirResponse{
			Data: guangyaclient.FSCreateDirData{
				FileID: "target-folder", FileName: "RKI-404-C", ParentID: "destination", ResType: 2,
			},
		},
		moveResult: guangyaclient.FileTaskResponse{
			Data: guangyaclient.FileTaskData{TaskID: "move-task"},
		},
		fileTaskStatus: guangyaclient.TaskStatusResponse{
			Data: guangyaclient.TaskStatusData{Status: 2, Progress: 100},
		},
	}
	service := newCloudJobTestService(t, client)
	job := cloudJob(t, false, "RKI-404-C.mp4", nil)
	job.SmartStrmStorageRoot = "/日本/有码/RKI"

	status, err := service.processCloudAddJob(&job)
	if err != nil {
		t.Fatalf("process job: %v", err)
	}
	if status != cloudJobCompleted || len(client.createDirNames) != 1 || client.createDirName != "RKI-404-C" {
		t.Fatalf("unexpected status or folder: status=%s folder=%s", status, client.createDirName)
	}
	if len(client.taskListIDs) != 1 || client.taskListIDs[0] != "cloud-task" {
		t.Fatalf("cloud task was not queried by task ID: %#v", client.taskListIDs)
	}
	if len(client.moveFileIDs) != 1 || client.moveFileIDs[0][0] != "movie" ||
		client.moveParents[0] != "target-folder" {
		t.Fatalf("unexpected move calls: ids=%#v parents=%#v", client.moveFileIDs, client.moveParents)
	}
}

func TestProcessCloudJobNotifiesSmartStrmWithOrganizedFolder(t *testing.T) {
	client := &fakeCloudClient{
		taskListResult: completedCloudTask("T28-205.mp4", false),
		filesByParent: map[string]guangyaclient.FileListResponse{
			"destination": fileList(guangyaclient.FileItem{
				FileID: "movie", FileName: "T28-205.mp4", ParentID: "destination", ResType: 1,
			}),
			"collection-folder": fileList(),
			"target-folder":     fileList(),
		},
		createDirResults: []guangyaclient.FSCreateDirResponse{
			{Data: guangyaclient.FSCreateDirData{
				FileID: "collection-folder", FileName: "T28", ParentID: "destination", ResType: 2,
			}},
			{Data: guangyaclient.FSCreateDirData{
				FileID: "target-folder", FileName: "T28-205", ParentID: "collection-folder", ResType: 2,
			}},
		},
		moveResult: guangyaclient.FileTaskResponse{
			Data: guangyaclient.FileTaskData{TaskID: "move-task"},
		},
		fileTaskStatus: guangyaclient.TaskStatusResponse{
			Data: guangyaclient.TaskStatusData{Status: 2, Progress: 100},
		},
	}
	service := newCloudJobTestService(t, client)
	strmRoot := t.TempDir()
	if err := service.db.Create(&model.SmartStrmConfig{
		URL: "http://smartstrm", Username: "tester", Password: "secret", StrmRootPath: strmRoot,
	}).Error; err != nil {
		t.Fatalf("create SmartStrm config: %v", err)
	}
	var actualTaskName string
	var actualStoragePath string
	service.runSmartStrm = func(_ model.SmartStrmConfig, taskName, storagePath string) error {
		actualTaskName = taskName
		actualStoragePath = storagePath
		return nil
	}
	job := cloudJob(t, false, "T28-205.mp4", nil)
	job.SmartStrmTaskName = "日本"
	job.SmartStrmStorageRoot = "/日本/有码"
	job.StrmSavePath = t.TempDir()
	if err := service.db.Create(&job).Error; err != nil {
		t.Fatalf("create cloud job: %v", err)
	}

	status, err := service.processCloudAddJob(&job)
	if err != nil {
		t.Fatalf("process job: %v", err)
	}
	if status != "" {
		t.Fatalf("status = %q, want pending", status)
	}
	if actualTaskName != "日本" || actualStoragePath != "/日本/有码/T28/T28-205" {
		t.Fatalf("unexpected SmartStrm call: task=%q path=%q", actualTaskName, actualStoragePath)
	}
	if !job.SmartStrmNotified || job.SmartStrmNotifiedAt == nil ||
		job.SmartStrmStoragePath != "/日本/有码/T28/T28-205" {
		t.Fatalf("SmartStrm notification was not persisted on the job: %#v", job)
	}
}

func TestProcessCloudJobDoesNotRetrySmartStrmNotificationAfterFailure(t *testing.T) {
	client := &fakeCloudClient{
		taskListResult: completedCloudTask("T28-205.mp4", false),
		filesByParent: map[string]guangyaclient.FileListResponse{
			"destination": fileList(guangyaclient.FileItem{
				FileID: "movie", FileName: "T28-205.mp4", ParentID: "destination", ResType: 1,
			}),
			"target-folder": fileList(),
		},
		createDirResult: guangyaclient.FSCreateDirResponse{
			Data: guangyaclient.FSCreateDirData{
				FileID: "target-folder", FileName: "T28-205", ParentID: "destination", ResType: 2,
			},
		},
		moveResult: guangyaclient.FileTaskResponse{
			Data: guangyaclient.FileTaskData{TaskID: "move-task"},
		},
		fileTaskStatus: guangyaclient.TaskStatusResponse{
			Data: guangyaclient.TaskStatusData{Status: 2, Progress: 100},
		},
	}
	service := newCloudJobTestService(t, client)
	if err := service.db.Create(&model.SmartStrmConfig{
		URL: "http://smartstrm", Username: "tester", Password: "secret",
	}).Error; err != nil {
		t.Fatalf("create SmartStrm config: %v", err)
	}
	calls := 0
	service.runSmartStrm = func(model.SmartStrmConfig, string, string) error {
		calls++
		return errors.New("request failed")
	}
	job := cloudJob(t, false, "T28-205.mp4", nil)
	job.SmartStrmTaskName = "日本"
	job.SmartStrmStorageRoot = "/日本/有码/T28"
	if err := service.db.Create(&job).Error; err != nil {
		t.Fatalf("create cloud job: %v", err)
	}

	status, err := service.processCloudAddJob(&job)
	if status != cloudJobFailed || err == nil {
		t.Fatalf("unexpected first attempt: status=%q err=%v", status, err)
	}
	if !job.SmartStrmNotified || calls != 1 {
		t.Fatalf("notification attempt was not persisted: notified=%v calls=%d", job.SmartStrmNotified, calls)
	}
	if finishErr := service.finishCloudJobAttempt(&job, status, err); finishErr != nil {
		t.Fatalf("finish failed attempt: %v", finishErr)
	}

	service.processPendingCloudJobs()
	if calls != 1 {
		t.Fatalf("SmartStrm notification was retried: calls=%d", calls)
	}
}

func TestProcessCloudJobWaitsForGeneratedStrmWithoutNotifyingAgain(t *testing.T) {
	service := newCloudJobTestService(t, &fakeCloudClient{})
	if err := service.db.Create(&model.SmartStrmConfig{
		URL: "http://smartstrm", StrmRootPath: t.TempDir(),
	}).Error; err != nil {
		t.Fatalf("create SmartStrm config: %v", err)
	}
	calls := 0
	service.runSmartStrm = func(model.SmartStrmConfig, string, string) error {
		calls++
		return nil
	}
	job := cloudJob(t, false, "T28-205.mp4", nil)
	job.SmartStrmNotified = true
	job.SmartStrmStorageRoot = "/日本/有码"
	job.SmartStrmStoragePath = "/日本/有码/T28/T28-205"
	job.StrmSavePath = t.TempDir()

	status, err := service.processCloudAddJob(&job)
	if err != nil {
		t.Fatalf("process job: %v", err)
	}
	if status != "" || job.CurrentStep != "等待 SmartStrm 生成 STRM" || job.Progress != 94 {
		t.Fatalf("unexpected waiting job: status=%q job=%#v", status, job)
	}
	if calls != 0 {
		t.Fatalf("SmartStrm was notified %d times while waiting", calls)
	}
}

func TestProcessCloudJobCopiesGeneratedStrm(t *testing.T) {
	t.Setenv("TRIM_PKGVAR", "/var/apps/fntube/var")
	service := newCloudJobTestService(t, &fakeCloudClient{})
	strmRoot := t.TempDir()
	saveRoot := t.TempDir()
	if err := service.db.Create(&model.SmartStrmConfig{
		URL: "http://smartstrm", StrmRootPath: strmRoot,
	}).Error; err != nil {
		t.Fatalf("create SmartStrm config: %v", err)
	}
	sourceDir := filepath.Join(strmRoot, "日本", "有码", "T28", "T28-205")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("create source directory: %v", err)
	}
	sourceContent := "http://example.test/T28-205.mp4"
	if err := os.WriteFile(filepath.Join(sourceDir, "T28-205.strm"), []byte(sourceContent), 0644); err != nil {
		t.Fatalf("write source strm: %v", err)
	}
	job := cloudJob(t, false, "T28-205.mp4", nil)
	job.SmartStrmNotified = true
	job.SmartStrmStorageRoot = "/日本/有码"
	job.SmartStrmStoragePath = "/日本/有码/T28/T28-205"
	job.StrmSavePath = saveRoot
	var searchKeyword string
	var scrapedGUID string
	service.SetMediaScrapeHooks(func(keyword string) ([]MediaSearchItem, error) {
		searchKeyword = keyword
		return []MediaSearchItem{
			{GUID: "partial-guid", Title: "T28-205"},
			{GUID: "other-guid", Title: "T28-205-CD1.strm"},
		}, nil
	}, func(guid string) error {
		scrapedGUID = guid
		return nil
	})

	status, err := service.processCloudAddJob(&job)
	if err != nil {
		t.Fatalf("process job: %v", err)
	}
	if status != "" || !job.StrmCopied || job.CurrentStep != "等待媒体库识别 STRM" || job.Progress != 98 {
		t.Fatalf("unexpected waiting job: status=%q job=%#v", status, job)
	}
	if searchKeyword != "T28-205" || scrapedGUID != "" {
		t.Fatalf("unexpected first media poll: keyword=%q scraped=%q", searchKeyword, scrapedGUID)
	}
	targetPath := filepath.Join(saveRoot, "T28", "T28-205", "T28-205.strm")
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read copied strm: %v", err)
	}
	if string(content) != sourceContent {
		t.Fatalf("copied content = %q, want %q", content, sourceContent)
	}

	if err := os.Remove(filepath.Join(sourceDir, "T28-205.strm")); err != nil {
		t.Fatalf("remove source strm: %v", err)
	}
	service.SetMediaScrapeHooks(func(keyword string) ([]MediaSearchItem, error) {
		searchKeyword = keyword
		return []MediaSearchItem{{GUID: "exact-guid", Title: " t28-205.STRM "}}, nil
	}, func(guid string) error {
		scrapedGUID = guid
		return nil
	})

	status, err = service.processCloudAddJob(&job)
	if err != nil {
		t.Fatalf("process copied job: %v", err)
	}
	if status != cloudJobCompleted || scrapedGUID != "exact-guid" {
		t.Fatalf("unexpected exact match result: status=%q scraped=%q", status, scrapedGUID)
	}
	if job.CurrentStep != "触发 STRM 媒体刮削" || job.Progress != 99 {
		t.Fatalf("unexpected scraping progress: %#v", job)
	}
}

func TestProcessCloudJobCopiesGeneratedStrmWithCollectionFolder(t *testing.T) {
	t.Setenv("TRIM_PKGVAR", "/var/apps/fntube/var")
	service := newCloudJobTestService(t, &fakeCloudClient{})
	strmRoot := t.TempDir()
	saveRoot := t.TempDir()
	if err := service.db.Create(&model.SmartStrmConfig{
		URL: "http://smartstrm", StrmRootPath: strmRoot,
	}).Error; err != nil {
		t.Fatalf("create SmartStrm config: %v", err)
	}
	sourceDir := filepath.Join(strmRoot, "日本", "有码", "RKI", "RKI-404")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("create source directory: %v", err)
	}
	sourceContent := "http://example.test/RKI-404.mp4"
	if err := os.WriteFile(filepath.Join(sourceDir, "RKI-404.strm"), []byte(sourceContent), 0644); err != nil {
		t.Fatalf("write source strm: %v", err)
	}
	job := cloudJob(t, false, "RKI-404.mp4", nil)
	job.SmartStrmNotified = true
	job.SmartStrmStorageRoot = "/日本/有码"
	job.SmartStrmStoragePath = "/日本/有码/RKI/RKI-404"
	job.StrmSavePath = saveRoot
	if err := service.db.Create(&job).Error; err != nil {
		t.Fatalf("create cloud job: %v", err)
	}
	var scrapedGUID string
	service.SetMediaScrapeHooks(func(keyword string) ([]MediaSearchItem, error) {
		if keyword != "RKI-404" {
			t.Fatalf("search keyword = %q, want RKI-404", keyword)
		}
		return []MediaSearchItem{{GUID: "rki-guid", Title: "RKI-404.strm"}}, nil
	}, func(guid string) error {
		scrapedGUID = guid
		return nil
	})

	status, err := service.processCloudAddJob(&job)
	if err != nil {
		t.Fatalf("process job: %v", err)
	}
	if status != cloudJobCompleted {
		t.Fatalf("status = %q, want %q", status, cloudJobCompleted)
	}
	if !job.StrmCopied || scrapedGUID != "rki-guid" {
		t.Fatalf("STRM was not copied and scraped: copied=%v guid=%q", job.StrmCopied, scrapedGUID)
	}
	var persisted model.GuangyaCloudAddJob
	if err := service.db.First(&persisted, job.ID).Error; err != nil {
		t.Fatalf("read persisted cloud job: %v", err)
	}
	if !persisted.StrmCopied {
		t.Fatal("copied STRM state was not persisted")
	}
	targetPath := filepath.Join(saveRoot, "RKI", "RKI-404", "RKI-404.strm")
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read copied strm: %v", err)
	}
	if string(content) != sourceContent {
		t.Fatalf("copied content = %q, want %q", content, sourceContent)
	}
}

func TestProcessCloudJobRejectsStrmCopyOutsideFnOS(t *testing.T) {
	t.Setenv("TRIM_PKGVAR", "")
	service := newCloudJobTestService(t, &fakeCloudClient{})
	strmRoot := t.TempDir()
	if err := service.db.Create(&model.SmartStrmConfig{
		URL: "http://smartstrm", StrmRootPath: strmRoot,
	}).Error; err != nil {
		t.Fatalf("create SmartStrm config: %v", err)
	}
	sourceDir := filepath.Join(strmRoot, "日本", "有码", "T28", "T28-205")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("create source directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "T28-205.strm"), []byte("movie"), 0644); err != nil {
		t.Fatalf("write source strm: %v", err)
	}
	job := cloudJob(t, false, "T28-205.mp4", nil)
	job.SmartStrmNotified = true
	job.SmartStrmStorageRoot = "/日本/有码"
	job.SmartStrmStoragePath = "/日本/有码/T28/T28-205"
	job.StrmSavePath = t.TempDir()

	status, err := service.processCloudAddJob(&job)
	if status != cloudJobFailed || err == nil || !strings.Contains(err.Error(), "不是飞牛环境") {
		t.Fatalf("unexpected result: status=%q err=%v", status, err)
	}
}

func TestProcessCloudJobRejectsStoragePathOutsideMapping(t *testing.T) {
	service := newCloudJobTestService(t, &fakeCloudClient{})
	if err := service.db.Create(&model.SmartStrmConfig{
		URL: "http://smartstrm", StrmRootPath: t.TempDir(),
	}).Error; err != nil {
		t.Fatalf("create SmartStrm config: %v", err)
	}
	job := cloudJob(t, false, "T28-205.mp4", nil)
	job.SmartStrmNotified = true
	job.SmartStrmStorageRoot = "/日本/有码"
	job.SmartStrmStoragePath = "/欧美/T28/T28-205"
	job.StrmSavePath = t.TempDir()

	status, err := service.processCloudAddJob(&job)
	if status != cloudJobFailed || err == nil || !strings.Contains(err.Error(), "不在存储映射目录") {
		t.Fatalf("unexpected result: status=%q err=%v", status, err)
	}
}

func TestProcessDirectoryRenamesSingleSelectedVideo(t *testing.T) {
	client := &fakeCloudClient{
		taskListResult: completedCloudTask("RKI-404-C", true),
		filesByParent: map[string]guangyaclient.FileListResponse{
			"destination": fileList(guangyaclient.FileItem{
				FileID: "root", FileName: "RKI-404-C", ParentID: "destination", ResType: 2,
			}),
			"root": fileList(guangyaclient.FileItem{
				FileID: "video", FileName: "original.mkv", FileSize: 1024, ParentID: "root", ResType: 1,
			}),
			"collection-folder": fileList(),
		},
		createDirResult: guangyaclient.FSCreateDirResponse{
			Data: guangyaclient.FSCreateDirData{
				FileID: "collection-folder", FileName: "RKI", ParentID: "destination", ResType: 2,
			},
		},
		moveResult: guangyaclient.FileTaskResponse{
			Data: guangyaclient.FileTaskData{TaskID: "move-task"},
		},
		fileTaskStatus: guangyaclient.TaskStatusResponse{
			Data: guangyaclient.TaskStatusData{Status: 2, Progress: 100},
		},
	}
	service := newCloudJobTestService(t, client)
	job := cloudJob(t, true, "RKI-404-C", []SelectedCloudFile{{
		FileIndex: 4, FileName: "original.mkv", FileSize: 1024, Path: []string{"source", "original.mkv"},
	}})

	status, err := service.processCloudAddJob(&job)
	if err != nil {
		t.Fatalf("process job: %v", err)
	}
	if status != cloudJobCompleted || len(client.moveFileIDs) != 1 ||
		len(client.renameNames) != 1 || client.renameNames[0] != "RKI-404-C.mkv" {
		t.Fatalf("unexpected operations: status=%s moves=%#v renames=%#v", status, client.moveFileIDs, client.renameNames)
	}
	if client.moveParents[0] != "collection-folder" || client.moveFileIDs[0][0] != "root" {
		t.Fatalf("movie folder was not moved into collection: ids=%#v parents=%#v", client.moveFileIDs, client.moveParents)
	}
}

func TestProcessDirectoryMovesNestedVideosAndNumbersNames(t *testing.T) {
	client := &fakeCloudClient{
		taskListResult: completedCloudTask("RKI-404-C", true),
		filesByParent: map[string]guangyaclient.FileListResponse{
			"destination": fileList(guangyaclient.FileItem{
				FileID: "root", FileName: "RKI-404-C", ParentID: "destination", ResType: 2,
			}),
			"root": fileList(guangyaclient.FileItem{
				FileID: "disc", FileName: "Disc", ParentID: "root", ResType: 2,
			}),
			"disc": fileList(
				guangyaclient.FileItem{
					FileID: "video-1", FileName: "part-a.mp4", FileSize: 100, ParentID: "disc", ResType: 1,
				},
				guangyaclient.FileItem{
					FileID: "video-2", FileName: "part-b.mkv", FileSize: 200, ParentID: "disc", ResType: 1,
				},
				guangyaclient.FileItem{
					FileID: "text", FileName: "readme.txt", ParentID: "disc", ResType: 1,
				},
			),
			"collection-folder": fileList(),
		},
		createDirResult: guangyaclient.FSCreateDirResponse{
			Data: guangyaclient.FSCreateDirData{
				FileID: "collection-folder", FileName: "RKI", ParentID: "destination", ResType: 2,
			},
		},
		moveResult: guangyaclient.FileTaskResponse{
			Data: guangyaclient.FileTaskData{TaskID: "move-task"},
		},
		fileTaskStatus: guangyaclient.TaskStatusResponse{
			Data: guangyaclient.TaskStatusData{Status: 2, Progress: 100},
		},
	}
	service := newCloudJobTestService(t, client)
	job := cloudJob(t, true, "RKI-404-C", []SelectedCloudFile{
		{FileIndex: 1, FileName: "part-a.mp4", FileSize: 100, Path: []string{"source", "Disc", "part-a.mp4"}},
		{FileIndex: 2, FileName: "part-b.mkv", FileSize: 200, Path: []string{"source", "Disc", "part-b.mkv"}},
	})

	status, err := service.processCloudAddJob(&job)
	if err != nil {
		t.Fatalf("process job: %v", err)
	}
	if status != cloudJobCompleted || len(client.moveFileIDs) != 3 {
		t.Fatalf("unexpected status or moves: status=%s moves=%#v", status, client.moveFileIDs)
	}
	if len(client.renameNames) != 2 ||
		client.renameNames[0] != "RKI-404-C1.mp4" ||
		client.renameNames[1] != "RKI-404-C2.mkv" {
		t.Fatalf("unexpected rename calls: %#v", client.renameNames)
	}
	for _, parent := range client.moveParents[:2] {
		if parent != "root" {
			t.Fatalf("nested file moved to %v, want root", parent)
		}
	}
	if client.moveParents[2] != "collection-folder" || client.moveFileIDs[2][0] != "root" {
		t.Fatalf("movie folder was not moved into collection: ids=%#v parents=%#v", client.moveFileIDs, client.moveParents)
	}
}

func TestCloudCollectionName(t *testing.T) {
	tests := map[string]string{
		"RKI-404-C.mp4": "RKI",
		"T28-205":       "T28",
		"n1901.mp4":     "N",
		"FC2-PPV-123":   "FC2",
	}
	for input, expected := range tests {
		if actual := cloudCollectionName(input); actual != expected {
			t.Errorf("cloudCollectionName(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestCloudAddJobPersistsProgressAndPublishesUpdates(t *testing.T) {
	service := newCloudJobTestService(t, &fakeCloudClient{})
	events, unsubscribe := service.SubscribeCloudJobs()
	defer unsubscribe()

	err := service.saveCloudAddJob("task-progress", CreateCloudTaskInput{
		ParentID: "destination",
		SelectedFiles: []SelectedCloudFile{{
			FileIndex: 1,
			FileName:  "movie.mp4",
		}},
	}, "MOVIE")
	if err != nil {
		t.Fatalf("save cloud add job: %v", err)
	}

	var created model.GuangyaCloudAddJob
	select {
	case event := <-events:
		created = event.Job
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for created event")
	}
	if created.CurrentStep != "任务已创建" || created.Progress != 5 || created.Status != cloudJobPending {
		t.Fatalf("unexpected created job: %#v", created)
	}

	if err := service.updateCloudJob(&created, map[string]any{
		"current_step": "整理文件",
		"progress":     80,
	}); err != nil {
		t.Fatalf("update cloud add job: %v", err)
	}
	select {
	case event := <-events:
		updated := event.Job
		if updated.CurrentStep != "整理文件" || updated.Progress != 80 {
			t.Fatalf("unexpected update event: %#v", updated)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for update event")
	}

	page, err := service.CloudAddJobs(1, 20)
	if err != nil {
		t.Fatalf("list cloud add jobs: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].TaskID != "task-progress" {
		t.Fatalf("unexpected cloud job page: %#v", page)
	}
}

func TestCloudJobProgressDoesNotDecrease(t *testing.T) {
	service := newCloudJobTestService(t, &fakeCloudClient{})
	job := cloudJob(t, false, "RKI-404.mp4", nil)
	job.Progress = 98

	if err := service.updateCloudJob(&job, map[string]any{
		"current_step": "等待 SmartStrm 生成 STRM",
		"progress":     94,
	}); err != nil {
		t.Fatalf("update cloud job: %v", err)
	}
	if job.Progress != 98 {
		t.Fatalf("progress decreased to %d", job.Progress)
	}
}

func TestDeleteCloudAddJobPersistsAndPublishesEvent(t *testing.T) {
	service := newCloudJobTestService(t, &fakeCloudClient{})
	job := cloudJob(t, false, "DELETE-ME.mp4", nil)
	job.TaskID = "delete-task"
	if err := service.db.Create(&job).Error; err != nil {
		t.Fatalf("create cloud job: %v", err)
	}
	events, unsubscribe := service.SubscribeCloudJobs()
	defer unsubscribe()

	if err := service.DeleteCloudAddJob(job.ID); err != nil {
		t.Fatalf("delete cloud job: %v", err)
	}
	select {
	case event := <-events:
		if event.Type != "deleted" || event.ID != job.ID {
			t.Fatalf("unexpected delete event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delete event")
	}

	var count int64
	if err := service.db.Model(&model.GuangyaCloudAddJob{}).Where("id = ?", job.ID).Count(&count).Error; err != nil {
		t.Fatalf("count deleted job: %v", err)
	}
	if count != 0 {
		t.Fatal("deleted cloud job still exists")
	}
	if err := service.DeleteCloudAddJob(job.ID); !errors.Is(err, ErrCloudJobNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestBackfillCloudJobProgressHandlesNullLegacyColumns(t *testing.T) {
	service := newCloudJobTestService(t, &fakeCloudClient{})
	now := time.Now()
	insert := `INSERT INTO guangya_cloud_add_jobs
		(task_id, requested_name, selected_files, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`
	if err := service.db.Exec(insert, "legacy-completed", "DONE", "[]", cloudJobCompleted, now, now).Error; err != nil {
		t.Fatalf("insert completed legacy job: %v", err)
	}
	if err := service.db.Exec(insert, "legacy-failed", "FAILED", "[]", cloudJobFailed, now, now).Error; err != nil {
		t.Fatalf("insert failed legacy job: %v", err)
	}

	service.backfillCloudJobProgress()

	var completed model.GuangyaCloudAddJob
	if err := service.db.Where("task_id = ?", "legacy-completed").First(&completed).Error; err != nil {
		t.Fatalf("read completed legacy job: %v", err)
	}
	if completed.CurrentStep != "已完成" || completed.Progress != 100 || completed.CompletedAt == nil {
		t.Fatalf("completed legacy job was not backfilled: %#v", completed)
	}

	var failed model.GuangyaCloudAddJob
	if err := service.db.Where("task_id = ?", "legacy-failed").First(&failed).Error; err != nil {
		t.Fatalf("read failed legacy job: %v", err)
	}
	if failed.CurrentStep != "云添加失败" || failed.CompletedAt == nil {
		t.Fatalf("failed legacy job was not backfilled: %#v", failed)
	}
}
