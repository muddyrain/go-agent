package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

const maxProjectFileBytes = 64 * 1024

var allowedProjectFileExtensions = map[string]struct{}{
	".go":   {},
	".md":   {},
	".txt":  {},
	".json": {},
	".yaml": {},
	".yml":  {},
	".toml": {},
	".mod":  {},
	".sum":  {},
}

type readProjectFileArguments struct {
	Path string `json:"path" jsonschema:"required,description=相对于项目根目录的文本文件路径"`
}

func newProjectFileTool(projectRoot string) (tool.InvokableTool, error) {
	return toolutils.InferTool(
		"read_project_file",
		"读取、查看、分析或总结 AgentHub 项目文件时使用；path 必须是相对于项目根目录的文本文件路径；禁止读取 .env；单个文件不能超过 64 KiB",
		func(
			ctx context.Context,
			input readProjectFileArguments,
		) (string, error) {
			if err := ctx.Err(); err != nil {
				return "", fmt.Errorf("context canceled: %w", err)
			}
			filePath, err := resolveProjectFilePath(
				projectRoot,
				input.Path,
			)
			if err != nil {
				return fmt.Sprintf(
					"无法读取项目文件：%v",
					err,
				), nil
			}
			content, err := readProjectTextFile(filePath)
			if err != nil {
				return fmt.Sprintf(
					"无法读取项目文件：%v",
					err,
				), nil
			}
			displayPath := filepath.ToSlash(
				filepath.Clean(strings.TrimSpace(input.Path)),
			)

			return fmt.Sprintf(
				"文件 %s 的内容：\n%s",
				displayPath,
				content,
			), nil
		},
	)
}

func resolveProjectFilePath(
	projectRoot string,
	requestedPath string,
) (string, error) {
	requestedPath = strings.TrimSpace(requestedPath)
	if requestedPath == "" {
		return "", fmt.Errorf("file path is required")
	}
	if filepath.IsAbs(requestedPath) {
		return "", fmt.Errorf("absolute file path is not allowed")
	}

	cleanPath := filepath.Clean(requestedPath)
	if cleanPath == "." {
		return "", fmt.Errorf("file path must point to a file")
	}
	if filepath.Base(cleanPath) == ".env" {
		return "", fmt.Errorf("reading .env is not allowed")
	}

	absoluteRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}

	resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return "", fmt.Errorf(
			"resolve project root symlinks: %w",
			err,
		)
	}

	targetPath := filepath.Join(resolvedRoot, cleanPath)

	// 先检查原始拼接路径，阻止普通的 ../ 路径穿越。
	if err := ensurePathInsideRoot(
		resolvedRoot,
		targetPath,
	); err != nil {
		return "", err
	}

	// 再解析符号链接，检查文件真正指向的位置。
	resolvedTarget, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		return "", fmt.Errorf("resolve file path: %w", err)
	}

	if err := ensurePathInsideRoot(
		resolvedRoot,
		resolvedTarget,
	); err != nil {
		return "", err
	}

	// 防止项目内符号链接指向项目外的 .env。
	if filepath.Base(resolvedTarget) == ".env" {
		return "", fmt.Errorf("reading .env is not allowed")
	}

	return resolvedTarget, nil
}

func ensurePathInsideRoot(
	projectRoot string,
	targetPath string,
) error {
	relativePath, err := filepath.Rel(projectRoot, targetPath)
	if err != nil {
		return fmt.Errorf("resolve relative file path: %w", err)
	}

	if relativePath == ".." ||
		strings.HasPrefix(
			relativePath,
			".."+string(filepath.Separator),
		) {
		return fmt.Errorf("file path escapes project root")
	}

	return nil
}

func readProjectTextFile(filePath string) (string, error) {
	extension := strings.ToLower(filepath.Ext(filePath))
	if _, ok := allowedProjectFileExtensions[extension]; !ok {
		return "", fmt.Errorf(
			"file extension %q is not allowed",
			extension,
		)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("inspect file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("path must point to a regular file")
	}
	if info.Size() > maxProjectFileBytes {
		return "", fmt.Errorf(
			"file exceeds the %d-byte limit",
			maxProjectFileBytes,
		)
	}

	content, err := io.ReadAll(
		io.LimitReader(file, maxProjectFileBytes+1),
	)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	if len(content) > maxProjectFileBytes {
		return "", fmt.Errorf(
			"file exceeds the %d-byte limit",
			maxProjectFileBytes,
		)
	}
	if !utf8.Valid(content) {
		return "", fmt.Errorf("file is not valid UTF-8 text")
	}

	return string(content), nil
}
