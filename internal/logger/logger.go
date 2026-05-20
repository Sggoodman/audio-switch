package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var global *zap.SugaredLogger

// Init 初始化全局 logger，日志写入 path 指定的文件。
// debug 为 true 时同时输出到 stdout。
func Init(path string, debug bool) {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	encoder := zapcore.NewConsoleEncoder(encoderCfg)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// 回退到 stdout
		core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)
		global = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()
		return
	}

	var cores []zapcore.Core
	cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(f), zapcore.DebugLevel))

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
