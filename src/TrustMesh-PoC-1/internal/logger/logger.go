package logger

import (
	"fmt"
	"os"
	"time"
)

// 日志级别常量
const (
	TEST    = "TEST"
	DEBUG   = "DEBUG"
	INFO    = "INFO"
	WARNING = "WARNING"
	ERROR   = "ERROR"
	FATAL   = "FATAL"
)

// 格式化输出函数
func log(level string, format string, a ...interface{}) {
	timestamp := time.Now().Format(time.RFC3339Nano)
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("[%s] [%s] %s\n", timestamp, level, msg)
}

// Test 日志，仅在开发时使用，发布时删除
func Test(format string, a ...interface{}) {
	if debugMode, isExist := os.LookupEnv("TEST_MODE"); !isExist || debugMode != "true" {
		return
	}

	log(TEST, format, a...)
}

// Debug 日志，仅在 DEBUG_MOD=true 时打印
func Debug(format string, a ...interface{}) {
	if debugMode, isExist := os.LookupEnv("DEBUG_MODE"); !isExist || debugMode != "true" {
		return
	}

	log(DEBUG, format, a...)
}

// Info 日志
func Info(format string, a ...interface{}) {
	log(INFO, format, a...)
}

// Warning 日志
func Warning(format string, a ...interface{}) {
	log(WARNING, format, a...)
}

// Error 日志
func Error(format string, a ...interface{}) {
	log(ERROR, format, a...)
}

// Fatal 日志，输出后立即退出程序
func Fatal(format string, a ...interface{}) {
	log(FATAL, format, a...)
	// 退出程序
	os.Exit(1)
}
