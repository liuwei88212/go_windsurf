package main

import (
	"GoTest/httpserver"
	"io"
	"log"
	"os"
)

func init() {
	// 设置日志格式，显示文件和行号，时间精确到毫秒
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	
	// 设置日志输出到文件
	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("无法创建日志文件:", err)
	}
	
	// 创建一个多重写入器，同时写入文件和标准输出
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
}

func main() {
	log.Println("服务器启动...")
	httpserver.StartServer()
}
