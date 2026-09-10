package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// loadLocalEnv 为本地开发可选加载 .env。
// godotenv.Load 只设置当前进程中尚未存在的变量，因此系统环境具有更高
// 优先级；生产环境可以完全通过进程环境注入配置，不需要部署 .env 文件。
func loadLocalEnv(path string) error {
	err := godotenv.Load(path)
	if err == nil {
		return nil
	}

	// .env 只是本地开发便利项，不是程序启动的必要文件；缺失时继续
	// 启动，之后由具体配置创建函数检查真正必需的环境变量。
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return fmt.Errorf("load local environment file %q: %w", path, err)
}
