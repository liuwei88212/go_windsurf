package main

import (
	"fmt"
	"log"
	"time"

	"GoTemplate/internal/app"
	"GoTemplate/internal/config"
)

func init() {
	// Set custom log format with millisecond precision
	log.SetFlags(0) // Clear default flags
	log.SetPrefix("")
	// Use custom logger that includes milliseconds
	defaultLogger := log.Default()
	defaultLogger.SetOutput(log.Writer())
	log.SetOutput(&logWriter{})
}

type logWriter struct{}

func (writer *logWriter) Write(bytes []byte) (int, error) {
	return fmt.Printf("%s %s", time.Now().Format("2006-01-02 15:04:05.000"), string(bytes))
}

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Failed to load config: %v\n", err)
		return
	}

	// 初始化应用
	app := app.New(cfg)
	log.Printf("Server starting on http://%s\n", cfg.Server.Address)

	// 启动服务
	if err := app.Run(); err != nil {
		log.Printf("Server failed to start: %v\n", err)
		return
	}
}
