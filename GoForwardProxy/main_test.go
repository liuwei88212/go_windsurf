package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestProxyBaidu 测试通过代理访问百度
func TestProxyBaidu(t *testing.T) {
	// 启动代理服务器
	go func() {
		proxy := NewForwardProxy()
		server := &http.Server{
			Addr:    ":8081", // 使用8081端口避免与主服务冲突
			Handler: proxy,
		}
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			t.Logf("代理服务器停止: %v", err)
		}
	}()

	// 等待代理服务器启动
	time.Sleep(time.Second)

	// Create proxy URL
	proxyURL, err := url.Parse("http://localhost:8081")
	if err != nil {
		t.Fatalf("解析代理URL失败: %v", err)
	}

	// 创建HTTP客户端
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 10 * time.Second,
	}

	// 测试用例
	tests := []struct {
		name       string
		targetURL  string
		wantStatus int
	}{
		{
			name:       "访问百度首页",
			targetURL:  "http://www.baidu.com",
			wantStatus: http.StatusOK,
		},
		{
			name:       "访问百度图片",
			targetURL:  "http://image.baidu.com",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 发送请求
			resp, err := client.Get(tt.targetURL)
			if err != nil {
				t.Fatalf("请求失败: %v", err)
			}
			defer resp.Body.Close()

			// 检查状态码
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("状态码不匹配, 期望 %d, 实际 %d", tt.wantStatus, resp.StatusCode)
			}

			// 读取响应内容
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("读取响应失败: %v", err)
			}

			// 检查响应内容是否包含特定关键字
			bodyStr := string(body)
			if !strings.Contains(bodyStr, "百度") {
				t.Error("响应内容中未找到'百度'关键字")
			}

			// 打印响应头信息
			fmt.Printf("响应头:\n")
			for key, values := range resp.Header {
				fmt.Printf("%s: %v\n", key, values)
			}

			t.Logf("成功访问 %s，响应长度: %d 字节", tt.targetURL, len(bodyStr))
		})
	}
}

// TestProxyHTTPS 测试HTTPS代理
func TestProxyHTTPS(t *testing.T) {
	// 启动代理服务器
	go func() {
		proxy := NewForwardProxy()
		server := &http.Server{
			Addr:    ":8082", // 使用8082端口
			Handler: proxy,
		}
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			t.Logf("代理服务器停止: %v", err)
		}
	}()

	// 等待代理服务器启动
	time.Sleep(time.Second)

	// 创建代理URL
	proxyURL, err := url.Parse("http://localhost:8082")
	if err != nil {
		t.Fatalf("解析代理URL失败: %v", err)
	}

	// 创建HTTP客户端
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			// 禁用证书验证，仅用于测试
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 10 * time.Second,
	}

	// 测试HTTPS请求
	resp, err := client.Get("https://www.baidu.com")
	if err != nil {
		t.Fatalf("HTTPS请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("HTTPS请求状态码不正确, 期望 %d, 实际 %d", http.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取HTTPS响应失败: %v", err)
	}

	t.Logf("HTTPS请求成功，响应长度: %d 字节", len(body))
}
