// Package db 封装数据库迁移、连接池配置和重试逻辑。
package db

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // PostgreSQL 驱动（匿名导入，注册驱动）
	_ "github.com/golang-migrate/migrate/v4/source/file"       // 文件源驱动（匿名导入，注册驱动）
)

// MigrateUp 执行所有未执行的迁移，把数据库升级到最新版本。
// migrationsPath 是迁移文件目录（如 "file://configs/migrations"）。
// databaseURL 是数据库连接字符串。
func MigrateUp(migrationsPath, databaseURL string) error {
	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()

	// Up() 执行所有未执行的迁移。
	// 如果已经是最新版本，返回 migrate.ErrNoChange（不是错误，正常情况）。
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations up: %w", err)
	}
	return nil
}
