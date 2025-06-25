package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func initLogger() *os.File {
	now := time.Now()
	logDir := "CInfoCollectLog"
	logFileName := fmt.Sprintf("%04d-%02d.log", now.Year(), int(now.Month()))
	logPath := filepath.Join(logDir, logFileName)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalln("无法创建日志目录:", err)
	}
	// 日志输出
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("无法打开日志文件:", err)
	}

	// 是否有控制台输出
	hasConsole := isTerminal(os.Stdout)
	if hasConsole {
		log.SetOutput(io.MultiWriter(os.Stdout, file))
	} else {
		log.SetOutput(file)
	}
	return file
}
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
