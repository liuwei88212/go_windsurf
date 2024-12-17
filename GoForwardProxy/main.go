package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

// ForwardProxy 正向代理服务器结构体
type ForwardProxy struct {
	client *http.Client
}

// NewForwardProxy 创建新的正向代理实例
func NewForwardProxy() *ForwardProxy {
	// 创建自定义的Transport
	transport := &http.Transport{
		// 设置代理服务器的TLS配置
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // 在生产环境中应该设置为false
		},
		// 设置连接池
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
		// 自定义拨号函数，支持超时设置
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	// 创建HTTP客户端
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Minute, // 整体请求超时时间
	}

	return &ForwardProxy{
		client: client,
	}
}

// ServeHTTP 处理代理请求
func (p *ForwardProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("收到请求: %s %s", r.Method, r.URL)

	if r.Method == http.MethodConnect {
		// 处理HTTPS请求
		p.handleHTTPS(w, r)
	} else {
		// 处理HTTP请求
		p.handleHTTP(w, r)
	}
}

// handleHTTP 处理HTTP请求
func (p *ForwardProxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	// 创建新的请求
	req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("创建请求失败: %v", err), http.StatusBadGateway)
		return
	}

	// 复制原始请求的header
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// 发送请求
	resp, err := p.client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("请求失败: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 复制响应header
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 写入状态码
	w.WriteHeader(resp.StatusCode)

	// 复制响应体
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("复制响应体失败: %v", err)
	}
}

// handleHTTPS 处理HTTPS请求
func (p *ForwardProxy) handleHTTPS(w http.ResponseWriter, r *http.Request) {
	// 劫持客户端连接
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "不支持的代理方式", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, fmt.Sprintf("连接劫持失败: %v", err), http.StatusServiceUnavailable)
		return
	}

	// 连接目标服务器
	targetConn, err := net.Dial("tcp", r.URL.Host)
	if err != nil {
		clientConn.Close()
		log.Printf("连接目标服务器失败: %v", err)
		return
	}

	// 发送200 Connection Established
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// 双向转发数据
	go transfer(targetConn, clientConn)
	go transfer(clientConn, targetConn)
}

// transfer 在两个连接之间转发数据
func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func main() {
	proxy := NewForwardProxy()
	server := &http.Server{
		Addr:    ":8080",
		Handler: proxy,
		// 设置超时
		ReadTimeout:    1 * time.Minute,
		WriteTimeout:   1 * time.Minute,
		IdleTimeout:    2 * time.Minute,
	}

	log.Printf("正向代理服务器启动在 :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
