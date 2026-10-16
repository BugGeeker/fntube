package systemtest

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type CopyResult struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Files       int    `json:"files"`
	Directories int    `json:"directories"`
	Bytes       int64  `json:"bytes"`
}

func CopyPath(source, destination string) (*CopyResult, error) {
	source = strings.TrimSpace(source)
	destination = strings.TrimSpace(destination)
	if source == "" || destination == "" {
		return nil, errors.New("源路径和目标路径不能为空")
	}

	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return nil, fmt.Errorf("解析源路径失败: %w", err)
	}
	destinationAbs, err := filepath.Abs(destination)
	if err != nil {
		return nil, fmt.Errorf("解析目标路径失败: %w", err)
	}

	info, err := os.Lstat(sourceAbs)
	if err != nil {
		return nil, fmt.Errorf("读取源路径失败: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("不支持复制符号链接")
	}

	finalDestination := destinationAbs
	if destinationInfo, statErr := os.Stat(destinationAbs); statErr == nil {
		if !destinationInfo.IsDir() {
			return nil, errors.New("目标文件已存在，不会覆盖")
		}
		finalDestination = filepath.Join(destinationAbs, filepath.Base(sourceAbs))
	} else if !os.IsNotExist(statErr) {
		return nil, fmt.Errorf("读取目标路径失败: %w", statErr)
	}

	if isSameOrNestedPath(sourceAbs, finalDestination) {
		return nil, errors.New("目标路径不能是源路径或源目录的子目录")
	}
	if _, err := os.Lstat(finalDestination); err == nil {
		return nil, errors.New("目标路径已存在，不会覆盖")
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("读取目标路径失败: %w", err)
	}

	result := &CopyResult{
		Source:      sourceAbs,
		Destination: finalDestination,
	}
	if info.IsDir() {
		if err := copyDirectory(sourceAbs, finalDestination, result); err != nil {
			return nil, err
		}
		return result, nil
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("源路径必须是普通文件或目录")
	}
	if err := copyFile(sourceAbs, finalDestination, info.Mode(), result); err != nil {
		return nil, err
	}
	return result, nil
}

func copyDirectory(source, destination string, result *CopyResult) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("目录中包含不支持的符号链接: %s", path)
		}
		if entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if err := os.Mkdir(target, info.Mode().Perm()); err != nil {
				return fmt.Errorf("创建目标目录失败 %s: %w", target, err)
			}
			result.Directories++
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("目录中包含不支持的文件类型: %s", path)
		}
		return copyFile(path, target, info.Mode(), result)
	})
}

func copyFile(source, destination string, mode fs.FileMode, result *CopyResult) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}
	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer sourceFile.Close()

	destinationFile, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm())
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	copied, copyErr := io.Copy(destinationFile, sourceFile)
	closeErr := destinationFile.Close()
	if copyErr != nil {
		_ = os.Remove(destination)
		return fmt.Errorf("复制文件失败: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(destination)
		return fmt.Errorf("写入目标文件失败: %w", closeErr)
	}
	result.Files++
	result.Bytes += copied
	return nil
}

func isSameOrNestedPath(source, destination string) bool {
	relative, err := filepath.Rel(source, destination)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}
