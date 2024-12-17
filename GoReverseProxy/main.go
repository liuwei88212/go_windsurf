package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// ProxyServer 代理服务器结构体
type ProxyServer struct {
	targetURL *url.URL        // 目标URL
	proxy    *httputil.ReverseProxy // 反向代理处理器
}

// LoggingMiddleware 日志中间件，用于记录请求日志
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("开始处理请求: %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("请求处理完成: %s %s，耗时: %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// NewProxyServer 创建新的代理服务器实例
func NewProxyServer(target string) (*ProxyServer, error) {
	// 解析目标URL
	targetURL, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("无效的目标URL: %v", err)
	}

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	
	// 自定义Director以修改转发前的请求
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// 添加自定义请求头
		req.Header.Set("X-Proxy-Server", "Go-Proxy")
		log.Printf("转发请求到: %s", targetURL.String())
	}

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("代理错误: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "代理服务器错误: %v", err)
	}

	return &ProxyServer{
		targetURL: targetURL,
		proxy:    proxy,
	}, nil
}

// ServeHTTP 处理代理请求
func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.proxy.ServeHTTP(w, r)
}

func main() {
	// 配置目标URL
	targetURL := "http://localhost:8081" // 示例目标地址
	
	// 创建代理服务器
	proxy, err := NewProxyServer(targetURL)
	if err != nil {
		log.Fatalf("创建代理服务器失败: %v", err)
	}

	// 创建路由器
	mux := http.NewServeMux()
	
	// 注册代理处理器并添加日志中间件
	mux.Handle("/", LoggingMiddleware(proxy))

	// 启动服务器
	serverAddr := ":8080"
	fmt.Printf("代理服务器启动于 %s，转发至 %s\n", serverAddr, targetURL)
	
	if err := http.ListenAndServe(serverAddr, mux); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
