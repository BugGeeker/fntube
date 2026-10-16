package guangya

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path"
	"strings"
	"time"

	guangyaclient "github.com/BugGeeker/guangyaclient-go"
	"gorm.io/gorm"

	"fntube/internal/model"
)

const (
	cloudJobPending   = "pending"
	cloudJobCompleted = "completed"
	cloudJobFailed    = "failed"
	cloudJobPollEvery = 10 * time.Second
)

func (s *Service) saveCloudAddJob(taskID string, input CreateCloudTaskInput, requestedName string) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return errors.New("光鸭云盘未返回任务 ID")
	}
	selected, err := json.Marshal(input.SelectedFiles)
	if err != nil {
		return err
	}
	job := model.GuangyaCloudAddJob{
		TaskID:              taskID,
		ParentID:            strings.TrimSpace(input.ParentID),
		RequestedName:       requestedName,
		ResolvedIsDirectory: input.ResolvedIsDirectory,
		SelectedFiles:       string(selected),
		Status:              cloudJobPending,
		CurrentStep:         "任务已创建",
		Progress:            5,
	}
	if err := s.db.Create(&job).Error; err != nil {
		return err
	}
	s.publishCloudJob(job)
	return nil
}

func (s *Service) StartCloudJobWorker() {
	s.workerOnce.Do(func() {
		s.backfillCloudJobProgress()
		go func() {
			s.processPendingCloudJobs()
			ticker := time.NewTicker(cloudJobPollEvery)
			defer ticker.Stop()
			for range ticker.C {
				s.processPendingCloudJobs()
			}
		}()
	})
}

func (s *Service) backfillCloudJobProgress() {
	if err := s.db.Model(&model.GuangyaCloudAddJob{}).
		Where("status = ? AND (progress IS NULL OR progress < ?)", cloudJobCompleted, 100).
		UpdateColumns(map[string]any{
			"current_step": "已完成",
			"progress":     100,
			"completed_at": gorm.Expr("COALESCE(completed_at, updated_at)"),
		}).Error; err != nil {
		log.Printf("[guangya] 补齐旧云添加完成记录失败: %v", err)
	}
	if err := s.db.Model(&model.GuangyaCloudAddJob{}).
		Where("status = ? AND (current_step IS NULL OR current_step = ?)", cloudJobFailed, "").
		UpdateColumns(map[string]any{
			"current_step": "云添加失败",
			"completed_at": gorm.Expr("COALESCE(completed_at, updated_at)"),
		}).Error; err != nil {
		log.Printf("[guangya] 补齐旧云添加失败记录失败: %v", err)
	}
}

func (s *Service) processPendingCloudJobs() {
	var jobs []model.GuangyaCloudAddJob
	if err := s.db.Where("status = ?", cloudJobPending).Order("id asc").Find(&jobs).Error; err != nil {
		log.Printf("[guangya] 读取云添加后处理任务失败: %v", err)
		return
	}
	for i := range jobs {
		status, err := s.processCloudAddJob(&jobs[i])
		if err != nil {
			log.Printf("[guangya] 云添加任务 %s 后处理失败: %v", jobs[i].TaskID, err)
		}
		if updateErr := s.finishCloudJobAttempt(&jobs[i], status, err); updateErr != nil {
			log.Printf("[guangya] 保存云添加任务 %s 状态失败: %v", jobs[i].TaskID, updateErr)
		}
	}
}

func (s *Service) processCloudAddJob(job *model.GuangyaCloudAddJob) (string, error) {
	task, found, err := s.cloudTask(job.TaskID)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("未查询到云添加任务")
	}
	switch task.Status {
	case 2:
		if err := s.updateCloudJob(job, map[string]any{
			"current_step": "云添加完成，正在整理文件",
			"progress":     75,
			"error":        "",
		}); err != nil {
			return "", err
		}
		var selected []SelectedCloudFile
		if err := json.Unmarshal([]byte(job.SelectedFiles), &selected); err != nil {
			return cloudJobFailed, fmt.Errorf("读取所选文件信息失败: %w", err)
		}
		if job.ResolvedIsDirectory {
			err = s.organizeDirectoryTask(*job, task, selected)
		} else {
			err = s.organizeSingleFileTask(*job, task)
		}
		if err != nil {
			return "", err
		}
		return cloudJobCompleted, nil
	case 3, 4:
		message := strings.TrimSpace(task.ErrMsg)
		if message == "" {
			message = fmt.Sprintf("云添加失败，任务状态为 %d", task.Status)
		}
		return cloudJobFailed, errors.New(message)
	default:
		step := "等待光鸭云盘处理"
		progress := 15
		if task.Status == 1 {
			step = "光鸭云盘正在添加文件"
			progress = 60
		}
		if err := s.updateCloudJob(job, map[string]any{
			"current_step": step,
			"progress":     progress,
			"error":        "",
		}); err != nil {
			return "", err
		}
		return "", nil
	}
}

func (s *Service) cloudTask(taskID string) (guangyaclient.CloudTask, bool, error) {
	result, err := cloudOperation(s, func(client cloudClient) (guangyaclient.CloudTaskListResponse, error) {
		return client.CloudTaskList(0, 10, []int{0, 1, 2, 3, 4}, []string{taskID})
	})
	if err != nil {
		return guangyaclient.CloudTask{}, false, err
	}
	for _, task := range result.Data.List {
		if task.TaskID == taskID {
			return task, true, nil
		}
	}
	return guangyaclient.CloudTask{}, false, nil
}

func (s *Service) organizeSingleFileTask(job model.GuangyaCloudAddJob, task guangyaclient.CloudTask) error {
	baseName := strings.TrimSuffix(job.RequestedName, path.Ext(job.RequestedName))
	if strings.TrimSpace(baseName) == "" {
		return errors.New("无法根据名称创建目标文件夹")
	}
	if err := s.updateCloudJob(&job, map[string]any{
		"current_step": "创建目标文件夹",
		"progress":     80,
	}); err != nil {
		return err
	}
	folder, err := s.ensureFolder(baseName, job.ParentID)
	if err != nil {
		return fmt.Errorf("创建目标文件夹失败: %w", err)
	}

	if err := s.updateCloudJob(&job, map[string]any{
		"current_step": "查找已添加文件",
		"progress":     86,
	}); err != nil {
		return err
	}
	fileNames := uniqueNonEmpty(job.RequestedName, task.FileName)
	if existing, ok, err := s.findDirectFile(folder.FileID, fileNames, false); err != nil {
		return err
	} else if ok {
		if existing.FileName != job.RequestedName {
			if err := s.updateCloudJob(&job, map[string]any{
				"current_step": "重命名文件",
				"progress":     95,
			}); err != nil {
				return err
			}
			return s.renameFile(existing.FileID, job.RequestedName)
		}
		return nil
	}

	file, ok, err := s.findDirectFile(job.ParentID, fileNames, false)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("未在保存目录找到已完成的文件 %q", job.RequestedName)
	}
	if file.FileName != job.RequestedName {
		if err := s.updateCloudJob(&job, map[string]any{
			"current_step": "重命名文件",
			"progress":     92,
		}); err != nil {
			return err
		}
		if err := s.renameFile(file.FileID, job.RequestedName); err != nil {
			return err
		}
	}
	if err := s.updateCloudJob(&job, map[string]any{
		"current_step": "移动文件到目标文件夹",
		"progress":     96,
	}); err != nil {
		return err
	}
	return s.moveFiles([]any{file.FileID}, folder.FileID)
}

func (s *Service) organizeDirectoryTask(job model.GuangyaCloudAddJob, task guangyaclient.CloudTask, selected []SelectedCloudFile) error {
	if len(selected) == 0 {
		return errors.New("没有可后处理的已选文件")
	}
	if err := s.updateCloudJob(&job, map[string]any{
		"current_step": "查找云添加目录",
		"progress":     80,
	}); err != nil {
		return err
	}
	rootNames := uniqueNonEmpty(task.FileName, job.RequestedName)
	root, ok, err := s.findDirectFile(job.ParentID, rootNames, true)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("未在保存目录找到云添加目录 %q", job.RequestedName)
	}

	if err := s.updateCloudJob(&job, map[string]any{
		"current_step": "扫描文件目录",
		"progress":     84,
	}); err != nil {
		return err
	}
	allFiles, err := s.walkFiles(root.FileID, nil)
	if err != nil {
		return err
	}
	used := make(map[string]bool)
	for index, wanted := range selected {
		fileProgress := 85 + (index * 12 / len(selected))
		if err := s.updateCloudJob(&job, map[string]any{
			"current_step": fmt.Sprintf("整理文件 %d/%d", index+1, len(selected)),
			"progress":     fileProgress,
		}); err != nil {
			return err
		}
		desiredName := desiredCloudFileName(job.RequestedName, index, len(selected), wanted.FileName)
		if existing := findFileByName(allFiles, desiredName, root.FileID, used); existing != nil {
			used[existing.item.FileID] = true
			continue
		}
		match := matchSelectedFile(allFiles, wanted, used)
		if match == nil {
			return fmt.Errorf("未找到已选文件 %q", strings.Join(wanted.Path, "/"))
		}
		used[match.item.FileID] = true
		if match.item.ParentID != root.FileID {
			if err := s.moveFiles([]any{match.item.FileID}, root.FileID); err != nil {
				return fmt.Errorf("移动文件 %q 失败: %w", match.item.FileName, err)
			}
		}
		if match.item.FileName != desiredName {
			if err := s.renameFile(match.item.FileID, desiredName); err != nil {
				return fmt.Errorf("重命名文件 %q 失败: %w", match.item.FileName, err)
			}
		}
	}
	return nil
}

type CloudJobPage struct {
	Total int64                      `json:"total"`
	Items []model.GuangyaCloudAddJob `json:"items"`
}

type CloudJobEvent struct {
	Type string
	Job  model.GuangyaCloudAddJob
	ID   uint
}

func (s *Service) CloudAddJobs(page, pageSize int) (CloudJobPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	var total int64
	if err := s.db.Model(&model.GuangyaCloudAddJob{}).Count(&total).Error; err != nil {
		return CloudJobPage{}, err
	}
	var jobs []model.GuangyaCloudAddJob
	err := s.db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&jobs).Error
	return CloudJobPage{Total: total, Items: jobs}, err
}

func (s *Service) DeleteCloudAddJob(id uint) error {
	result := s.db.Delete(&model.GuangyaCloudAddJob{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCloudJobNotFound
	}
	s.publishCloudJobEvent(CloudJobEvent{Type: "deleted", ID: id})
	return nil
}

func (s *Service) ActiveCloudAddJobs() ([]model.GuangyaCloudAddJob, error) {
	var jobs []model.GuangyaCloudAddJob
	err := s.db.Where("status = ?", cloudJobPending).Order("id desc").Find(&jobs).Error
	return jobs, err
}

func (s *Service) SubscribeCloudJobs() (<-chan CloudJobEvent, func()) {
	s.subscriberMu.Lock()
	id := s.nextSubscriber
	s.nextSubscriber++
	ch := make(chan CloudJobEvent, 16)
	s.subscribers[id] = ch
	s.subscriberMu.Unlock()

	return ch, func() {
		s.subscriberMu.Lock()
		delete(s.subscribers, id)
		s.subscriberMu.Unlock()
	}
}

func (s *Service) publishCloudJob(job model.GuangyaCloudAddJob) {
	s.publishCloudJobEvent(CloudJobEvent{Type: "progress", Job: job, ID: job.ID})
}

func (s *Service) publishCloudJobEvent(event CloudJobEvent) {
	s.subscriberMu.Lock()
	defer s.subscriberMu.Unlock()
	for _, ch := range s.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func (s *Service) updateCloudJob(job *model.GuangyaCloudAddJob, updates map[string]any) error {
	if progress, ok := updates["progress"].(int); ok {
		if progress < 0 {
			updates["progress"] = 0
		} else if progress > 100 {
			updates["progress"] = 100
		}
	}
	if job.ID == 0 {
		applyCloudJobUpdates(job, updates)
		return nil
	}
	if err := s.db.Model(job).Updates(updates).Error; err != nil {
		return err
	}
	if err := s.db.First(job, job.ID).Error; err != nil {
		return err
	}
	s.publishCloudJob(*job)
	return nil
}

func (s *Service) finishCloudJobAttempt(job *model.GuangyaCloudAddJob, status string, attemptErr error) error {
	updates := make(map[string]any)
	if status != "" {
		updates["status"] = status
	}
	if attemptErr != nil {
		updates["error"] = attemptErr.Error()
		if status == cloudJobFailed {
			now := s.now()
			updates["current_step"] = "云添加失败"
			updates["completed_at"] = &now
		} else {
			updates["current_step"] = "处理异常，等待重试"
			if job.Progress < 5 {
				updates["progress"] = 5
			}
		}
	} else if status == cloudJobCompleted {
		now := s.now()
		updates["error"] = ""
		updates["current_step"] = "已完成"
		updates["progress"] = 100
		updates["completed_at"] = &now
	}
	if len(updates) == 0 {
		return nil
	}
	return s.updateCloudJob(job, updates)
}

func applyCloudJobUpdates(job *model.GuangyaCloudAddJob, updates map[string]any) {
	if value, ok := updates["status"].(string); ok {
		job.Status = value
	}
	if value, ok := updates["current_step"].(string); ok {
		job.CurrentStep = value
	}
	if value, ok := updates["progress"].(int); ok {
		job.Progress = value
	}
	if value, ok := updates["error"].(string); ok {
		job.Error = value
	}
	if value, ok := updates["completed_at"].(*time.Time); ok {
		job.CompletedAt = value
	}
}

func (s *Service) ensureFolder(name, parentID string) (CloudFolder, error) {
	if existing, ok, err := s.findDirectFile(parentID, []string{name}, true); err != nil {
		return CloudFolder{}, err
	} else if ok {
		return CloudFolder{
			FileID: existing.FileID, FileName: existing.FileName,
			ResType: existing.ResType, ParentID: existing.ParentID,
		}, nil
	}
	return s.CreateFolder(name, parentID)
}

func (s *Service) findDirectFile(parentID string, names []string, directory bool) (guangyaclient.FileItem, bool, error) {
	items, err := s.listAllFiles(parentID)
	if err != nil {
		return guangyaclient.FileItem{}, false, err
	}
	for _, name := range names {
		for _, item := range items {
			if item.FileName == name && (item.ResType == 2) == directory {
				return item, true, nil
			}
		}
	}
	return guangyaclient.FileItem{}, false, nil
}

func (s *Service) listAllFiles(parentID string) ([]guangyaclient.FileItem, error) {
	const pageSize = 100
	var all []guangyaclient.FileItem
	for pageNumber := 0; pageNumber < 1000; pageNumber++ {
		result, err := cloudOperation(s, func(client cloudClient) (guangyaclient.FileListResponse, error) {
			return client.FSFiles(parentID, pageNumber, pageSize, 0, 0, nil, nil, nil, false)
		})
		if err != nil {
			return nil, err
		}
		all = append(all, result.Data.List...)
		if len(result.Data.List) == 0 || len(all) >= result.Data.Total {
			return all, nil
		}
	}
	return nil, errors.New("光鸭目录文件数量超过处理上限")
}

type walkedCloudFile struct {
	item guangyaclient.FileItem
	path []string
}

func (s *Service) walkFiles(parentID string, parentPath []string) ([]walkedCloudFile, error) {
	items, err := s.listAllFiles(parentID)
	if err != nil {
		return nil, err
	}
	var result []walkedCloudFile
	for _, item := range items {
		itemPath := appendPath(parentPath, item.FileName)
		if item.ResType == 2 {
			children, err := s.walkFiles(item.FileID, itemPath)
			if err != nil {
				return nil, err
			}
			result = append(result, children...)
			continue
		}
		result = append(result, walkedCloudFile{item: item, path: itemPath})
	}
	return result, nil
}

func (s *Service) moveFiles(fileIDs []any, parentID string) error {
	result, err := cloudOperation(s, func(client cloudClient) (guangyaclient.FileTaskResponse, error) {
		return client.FSMove(fileIDs, parentID)
	})
	if err != nil {
		return err
	}
	if strings.TrimSpace(result.Data.TaskID) == "" {
		return nil
	}
	for attempt := 0; attempt < 60; attempt++ {
		status, err := cloudOperation(s, func(client cloudClient) (guangyaclient.TaskStatusResponse, error) {
			return client.GetTaskStatus(result.Data.TaskID)
		})
		if err != nil {
			return err
		}
		if status.Data.Status == 2 {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return errors.New("等待光鸭文件移动完成超时")
}

func (s *Service) renameFile(fileID, name string) error {
	_, err := cloudOperation(s, func(client cloudClient) (guangyaclient.EmptyResponse, error) {
		return client.FSRename(fileID, name)
	})
	return err
}

func desiredCloudFileName(baseName string, index, total int, originalName string) string {
	suffix := ""
	if total > 1 {
		suffix = fmt.Sprintf("%d", index+1)
	}
	return baseName + suffix + path.Ext(originalName)
}

func matchSelectedFile(files []walkedCloudFile, wanted SelectedCloudFile, used map[string]bool) *walkedCloudFile {
	var fallback *walkedCloudFile
	for i := range files {
		file := &files[i]
		if used[file.item.FileID] || !strings.EqualFold(file.item.FileName, wanted.FileName) {
			continue
		}
		if wanted.FileSize > 0 && file.item.FileSize > 0 && wanted.FileSize != file.item.FileSize {
			continue
		}
		if pathsMatch(file.path, wanted.Path) {
			return file
		}
		if fallback == nil {
			fallback = file
		}
	}
	return fallback
}

func findFileByName(files []walkedCloudFile, name, parentID string, used map[string]bool) *walkedCloudFile {
	for i := range files {
		if !used[files[i].item.FileID] && files[i].item.ParentID == parentID && files[i].item.FileName == name {
			return &files[i]
		}
	}
	return nil
}

func pathsMatch(left, right []string) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	shorter := len(left)
	if len(right) < shorter {
		shorter = len(right)
	}
	for offset := 1; offset <= shorter; offset++ {
		if !strings.EqualFold(left[len(left)-offset], right[len(right)-offset]) {
			return false
		}
	}
	return true
}

func appendPath(parent []string, name string) []string {
	result := make([]string, len(parent), len(parent)+1)
	copy(result, parent)
	return append(result, name)
}

func uniqueNonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		found := false
		for _, existing := range result {
			if existing == value {
				found = true
				break
			}
		}
		if !found {
			result = append(result, value)
		}
	}
	return result
}
