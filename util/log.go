package util

import (
	"log/slog"
	"os"
)

var Log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	AddSource: true,            // 是否输出所在文件机器行数
	Level:     slog.LevelDebug, // 输出最低级别，低于该级别的日志将会被丢弃
}))
