package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// 模拟后端服务器
func mockBackendServer() *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 记录收到的请求头
		proxyHeader := r.Header.Get("X-Proxy-Server")
		// 返回一个测试响应
		fmt.Fprintf(w, "测试响应：收到请求路径 %s，代理头：%s", r.URL.Path, proxyHeader)
	})
	return httptest.NewServer(handler)
}

// TestProxyServer 测试代理服务器的基本功能
func TestProxyServer(t *testing.T) {
	// 启动模拟的后端服务器
	backendServer := mockBackendServer()
	defer backendServer.Close()

	// 创建代理服务器
	proxy, err := NewProxyServer(backendServer.URL)
	if err != nil {
		t.Fatalf("创建代理服务器失败: %v", err)
	}

	// 创建测试服务器
	proxyServer := httptest.NewServer(LoggingMiddleware(proxy))
	defer proxyServer.Close()

	// 创建测试用例
	testCases := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "测试根路径",
			path:           "/",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "测试API路径",
			path:           "/api/test",
			expectedStatus: http.StatusOK,
		},
	}

	// 运行测试用例
	client := &http.Client{Timeout: 5 * time.Second}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 发送请求
			resp, err := client.Get(proxyServer.URL + tc.path)
			if err != nil {
				t.Fatalf("请求失败: %v", err)
			}
			defer resp.Body.Close()

			// 检查状态码
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("状态码不匹配，期望 %d，实际 %d", tc.expectedStatus, resp.StatusCode)
			}

			// 读取响应内容
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("读取响应失败: %v", err)
			}

			// 检查响应中是否包含代理头信息
			bodyStr := string(body)
			if !strings.Contains(bodyStr, "代理头：Go-Proxy") {
				t.Errorf("响应中未找到预期的代理头信息: %s", bodyStr)
			}

			t.Logf("响应内容: %s", bodyStr)
		})
	}
}

// TestProxyServerError 测试代理服务器错误处理
func TestProxyServerError(t *testing.T) {
	// 创建一个指向不存在服务器的代理
	proxy, err := NewProxyServer("http://localhost:44444")
	if err != nil {
		t.Fatalf("创建代理服务器失败: %v", err)
	}

	// 创建测试服务器
	proxyServer := httptest.NewServer(LoggingMiddleware(proxy))
	defer proxyServer.Close()

	// 发送请求
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(proxyServer.URL)
	if err != nil {
		t.Logf("预期的错误发生: %v", err)
		return
	}
	defer resp.Body.Close()

	// 检查是否返回502错误
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusBadGateway, resp.StatusCode)
	}
}
