package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// isRetryable 判断数据库错误是否可以重试。
// 临时性错误（连接断开、序列化失败、死锁）可以重试；
// 永久性错误（唯一约束、外键、语法错误）不能重试。
func isRetryable(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		// 不是 PostgreSQL 错误（比如网络超时），保守地认为可以重试
		return true
	}
	switch pgErr.Code {
	case "08006", // 连接失败
		"40001", // 序列化失败（事务冲突，重试可能成功）
		"40P01", // 死锁检测
		"57P01", // 管理员终止连接
		"57P03": // 无法连接（数据库启动中）
		return true
	default:
		return false
	}
}

// Retry 执行 fn，遇到可重试错误时自动重试，最多 attempts 次。
// 重试间隔用指数退避：100ms → 200ms → 400ms → ...
// ctx 被取消时立即返回。
func Retry(ctx context.Context, attempts int, fn func() error) error {
	if attempts < 1 {
		attempts = 1
	}
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if !isRetryable(err) {
			return err // 不可重试，直接返回
		}
		if i == attempts-1 {
			break // 最后一次尝试失败，不再等待
		}
		// 指数退避：第 i 次等待 100 * 2^i 毫秒
		wait := time.Duration(100*(1<<i)) * time.Millisecond
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return err
}
