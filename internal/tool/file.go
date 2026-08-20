package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ListDirTool struct {
	WorkDir string
}

type ReadFileTool struct {
	WorkDir string
}

type SearchCodeTool struct {
	WorkDir string
}

// safePath 把用户传入的相对路径解析为工作目录内的绝对路径
// 返回解析后的路径，如果路径越界则返回错误
func safePath(workDir, userPath string) (string, error) {
	// 拼接为绝对路径
	absPath := filepath.Join(workDir, userPath)

	// 清理路径（解析 . 和 ..）
	absPath = filepath.Clean(absPath)

	// 确保在工作目录内
	rel, err := filepath.Rel(workDir, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path %q is outside working directory", userPath)
	}

	return absPath, nil
}

func (t *ListDirTool) Name() string {
	return "list_directory"
}

func (t *ListDirTool) Description() string {
	return "列出指定目录下的文件和子目录，路径相对于项目根目录"
}

func (t *ListDirTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
        "type": "object",
        "properties": {
            "path": {"type": "string", "description": "要列出的目录路径，默认为根目录"}
        },
        "required": []
    }`)
}

func (t *ListDirTool) Execute(ctx context.Context, arguments json.RawMessage) (string, error) {
	// 解析参数
	var params struct {
		Path string `json:"path"`
	}

	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}
	if params.Path == "" {
		params.Path = "."
	}
	// 安全校验
	absPath, err := safePath(t.WorkDir, params.Path)
	if err != nil {
		return "", err
	}

	// 读取目录
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return "", fmt.Errorf("read directory: %w", err)
	}

	// 格式化输出
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("目录: %s\n", params.Path))

	for _, entry := range entries {
		if entry.IsDir() {
			sb.WriteString(fmt.Sprintf("  [DIR]  %s/\n", entry.Name()))
		} else {
			info, _ := entry.Info()
			sb.WriteString(fmt.Sprintf("  [FILE] %s (%d bytes)\n", entry.Name(), info.Size()))
		}
	}
	return sb.String(), nil
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "读取指定文件的内容，路径相对于项目根目录"
}

func (t *ReadFileTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
        "type": "object",
        "properties": {
            "path": {"type": "string", "description": "要读取的文件路径"}
        },
        "required": ["path"]
    }`)
}

func (t *ReadFileTool) Execute(ctx context.Context, arguments json.RawMessage) (string, error) {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(arguments, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if params.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	absPath, err := safePath(t.WorkDir, params.Path)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return fmt.Sprintf("文件: %s\n---\n%s\n---", params.Path, string(content)), nil
}

func (t *SearchCodeTool) Name() string {
	return "search_code"
}

func (t *SearchCodeTool) Description() string {
	return "在项目文件中搜索关键词，返回匹配的文件名和行号"
}

func (t *SearchCodeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
        "type": "object",
        "properties": {
            "pattern": {"type": "string", "description": "要搜索的关键词"},
            "path": {"type": "string", "description": "搜索的起始目录，默认为根目录"}
        },
        "required": ["pattern"]
    }`)
}

func (t *SearchCodeTool) Execute(ctx context.Context, arguments json.RawMessage) (string, error) {
	var params struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal(arguments, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if params.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}
	if params.Path == "" {
		params.Path = "."
	}
	absPath, err := safePath(t.WorkDir, params.Path)
	if err != nil {
		return "", err
	}
	var results []string
	// 遍历目录
	err = filepath.WalkDir(absPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // 跳过无权访问的文件
		}
		if d.IsDir() {
			// 跳过 .git 等隐藏目录
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		// 只搜文本文件（简单判断后缀）
		if !isTextFile(d.Name()) {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// 逐行匹配
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if strings.Contains(line, params.Pattern) {
				relPath, _ := filepath.Rel(t.WorkDir, path)
				results = append(results, fmt.Sprintf("%s:%d: %s", relPath, i+1, strings.TrimSpace(line)))
			}
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		return fmt.Sprintf("未找到包含 %q 的内容", params.Pattern), nil
	}

	return fmt.Sprintf("搜索 %q 的结果（共 %d 条）：\n%s",
		params.Pattern, len(results), strings.Join(results, "\n")), nil
}

// isTextFile 简单判断是否为文本文件
func isTextFile(name string) bool {
	textExts := []string{".go", ".md", ".txt", ".json", ".yaml", ".yml", ".toml", ".mod", ".sum", ".html", ".css", ".js", ".ts"}
	for _, ext := range textExts {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}
