package obs

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Logger 是最小结构化日志器
type Logger struct{}

// Field 是一个结构化日志字段（键值对）
type Field struct {
	Key   string
	Value any
}

// New 创建一个新的 Logger，输出到 stdout
func New() *Logger {
	return &Logger{}
}

// String 创建一个字符串类型的 Field
func String(key, val string) Field {
	return Field{Key: key, Value: val}
}

// Int 创建一个整数类型的 Field
func Int(key string, val int) Field {
	return Field{Key: key, Value: val}
}

// Info 输出一条 info 级别日志
func (l *Logger) Info(msg string, fields ...Field) {
	l.log("info", msg, fields)
}

// Error 输出一条 error 级别日志
func (l *Logger) Error(msg string, fields ...Field) {
	l.log("error", msg, fields)
}

func (l *Logger) log(level, msg string, fields []Field) {
	// 基础字段：level、时间戳、消息
	m := map[string]any{
		"level": level,
		"ts":    time.Now().Format(time.RFC3339),
		"msg":   msg,
	}

	// 把用户传的字段展开到 map 里
	for _, f := range fields {
		m[f.Key] = f.Value
	}

	// 序列化为 JSON
	b, err := json.Marshal(m)
	if err != nil {
		// 序列化失败就降级打纯文本，不能因为日志问题搞崩程序
		fmt.Fprintf(os.Stderr, "log marshal error: %v, msg=%s\n", err, msg)
		return
	}

	// 输出到 stdout
	fmt.Println(string(b))
}
