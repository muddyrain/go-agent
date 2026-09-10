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

// 文件工具只开放少量 UTF-8 文本格式，并限制单文件大小。
// 这是模型可读取内容的能力边界，不等同于操作系统文件权限。
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

// readProjectFileArguments 同时定义 Go 输入结构和模型可见的 JSON Schema；
// InferTool 会根据 json/jsonschema 标签生成 path 参数说明并反序列化调用参数。
type readProjectFileArguments struct {
	Path string `json:"path" jsonschema:"required,description=相对于项目根目录的文本文件路径"`
}

// newProjectFileTool 创建受项目根目录约束的本地只读工具。
// 路径校验和文件读取留在普通 Go 函数中，便于独立测试安全规则，而不是
// 把所有逻辑塞进 Eino Tool Handler。
func newProjectFileTool(projectRoot string) (tool.InvokableTool, error) {
	return toolutils.InferTool(
		"read_project_file",
		"读取、查看、分析或总结 AgentHub 项目文件时使用；path 必须是相对于项目根目录的文本文件路径；禁止读取 .env；单个文件不能超过 64 KiB",
		func(
			ctx context.Context,
			input readProjectFileArguments,
		) (string, error) {
			// Context 取消代表调用生命周期已经结束，必须返回真正的 Go error，
			// 让 Eino 停止当前执行；不能把它包装成模型可继续处理的 ToolResult。
			if err := ctx.Err(); err != nil {
				return "", fmt.Errorf("context canceled: %w", err)
			}
			filePath, err := resolveProjectFilePath(
				projectRoot,
				input.Path,
			)
			if err != nil {
				// 路径不满足工具策略属于一次可解释的业务拒绝：返回文字且
				// error 为 nil，使 ToolResult 能进入消息历史并由模型向用户解释。
				return fmt.Sprintf(
					"无法读取项目文件：%v",
					err,
				), nil
			}
			content, err := readProjectTextFile(filePath)
			if err != nil {
				// 文件类型、大小、普通文件和 UTF-8 检查失败同样属于
				// 可反馈给模型的工具结果，而不是 Agent 执行框架故障。
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

// resolveProjectFilePath 把模型提供的相对路径解析成项目内真实路径。
// 它先阻止普通 ../ 穿越，再解析符号链接检查最终目标，二者缺一不可。
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

// ensurePathInsideRoot 使用 filepath.Rel 判断 target 是否仍位于 root 内。
// 不能只做字符串前缀比较，例如 /project-other 会错误匹配 /project。
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

// readProjectTextFile 只读取允许类型、普通文件、限制大小且编码有效的文本。
// 路径是否位于项目根目录由 resolveProjectFilePath 负责，本函数只处理
// 已解析文件本身的内容约束。
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
