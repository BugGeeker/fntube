package guangya

import (
	"encoding/json"
	"errors"
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

	status, err := service.processCloudAddJob(&job)
	if err != nil {
		t.Fatalf("process job: %v", err)
	}
	if status != cloudJobCompleted || client.createDirName != "RKI-404-C" {
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
	if status != cloudJobCompleted || len(client.moveFileIDs) != 0 ||
		len(client.renameNames) != 1 || client.renameNames[0] != "RKI-404-C.mkv" {
		t.Fatalf("unexpected operations: status=%s moves=%#v renames=%#v", status, client.moveFileIDs, client.renameNames)
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
	if status != cloudJobCompleted || len(client.moveFileIDs) != 2 {
		t.Fatalf("unexpected status or moves: status=%s moves=%#v", status, client.moveFileIDs)
	}
	if len(client.renameNames) != 2 ||
		client.renameNames[0] != "RKI-404-C1.mp4" ||
		client.renameNames[1] != "RKI-404-C2.mkv" {
		t.Fatalf("unexpected rename calls: %#v", client.renameNames)
	}
	for _, parent := range client.moveParents {
		if parent != "root" {
			t.Fatalf("nested file moved to %v, want root", parent)
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
