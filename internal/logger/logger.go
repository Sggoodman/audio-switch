package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var global *zap.SugaredLogger

func init() {
	// 默认初始化到 stdout，确保 global 不为 nil
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(newEncoderConfig()),
		zapcore.AddSync(os.Stdout),
		zapcore.DebugLevel,
	)
	global = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()
}

func newEncoderConfig() zapcore.EncoderConfig {
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "ts"
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncodeLevel = zapcore.CapitalLevelEncoder
	return cfg
}

// Init 初始化全局 logger，日志写入 path 指定的文件。
// debug 为 true 时同时输出到 stdout。文件输出自动轮转（单文件最大 5MB，保留 3 个备份）。
func Init(path string, debug bool) {
	encoder := zapcore.NewConsoleEncoder(newEncoderConfig())

	writer := &lumberjack.Logger{
		Filename: filepath.Join(filepath.Dir(path), filepath.Base(path)),
		MaxSize:  5, // MB
		MaxBackups: 3,
	}

	var cores []zapcore.Core
	cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(writer), zapcore.DebugLevel))

	if debug {
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel))
	}

	core := zapcore.NewTee(cores...)
	global = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()
}

// Sync 刷写缓冲，程序退出前调用。
func Sync() {
	if global != nil {
		_ = global.Sync()
	}
}

// Debug 输出调试级别日志。
func Debug(module, msg string, keysAndValues ...any) {
	global.Debugw(msg, append([]any{"module", module}, keysAndValues...)...)
}

// Info 输出信息级别日志。
func Info(module, msg string, keysAndValues ...any) {
	global.Infow(msg, append([]any{"module", module}, keysAndValues...)...)
}

// Warn 输出警告级别日志。
func Warn(module, msg string, keysAndValues ...any) {
	global.Warnw(msg, append([]any{"module", module}, keysAndValues...)...)
}

// Error 输出错误级别日志。
func Error(module, msg string, keysAndValues ...any) {
	global.Errorw(msg, append([]any{"module", module}, keysAndValues...)...)
}
