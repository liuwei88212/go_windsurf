package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/valyala/fasthttp"
)

var connections = make(map[string]net.Conn)

// handleOpenConnection 处理打开到指定URL的TCP连接的HTTP请求
func handleOpenConnection(ctx *fasthttp.RequestCtx) {
	// 检查请求方法是否为POST
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("只允许POST方法", fasthttp.StatusMethodNotAllowed)
		return
	}

	// 从请求体中读取目标URL
	targetURL := strings.TrimSpace(string(ctx.PostBody()))

	host, _, _, err := parseURLStr(targetURL)
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}
	// 建立TCP连接
	conn, err := net.Dial("tcp", host)
	if err != nil {
		ctx.Error("无法建立TCP连接", fasthttp.StatusInternalServerError)
		return
	}

	// 存储连接
	connections[host] = conn

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody([]byte("TCP连接已建立"))

	log.Printf("TCP连接已建立到 %s", host)
}

// assembleHTTPRequest constructs an HTTP request string to be sent over a TCP connection.
func assembleHTTPRequest(method, host, path, params string, body []byte) string {
	requestLine := fmt.Sprintf("%s %s?%s HTTP/1.1\r\n", method, path, params)
	headers := fmt.Sprintf("Host: %s\r\nContent-Length: %d\r\n\r\n", host, len(body))
	return requestLine + headers + string(body)
}

// handleSendRequest 处理发送HTTP请求的接口
func handleSendRequest(ctx *fasthttp.RequestCtx) {
	// 检查请求方法是否为POST
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("只允许POST方法", fasthttp.StatusMethodNotAllowed)
		return
	}

	// 从请求体中读取目标URL和请求数据
	targetURL := string(ctx.QueryArgs().Peek("targetURL"))

	host, path, _, err := parseURLStr(targetURL)
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}

	requestData := ctx.PostBody()

	// 获取已建立的TCP连接
	conn, exists := connections[host]
	if !exists {
		ctx.Error("未找到已建立的连接", fasthttp.StatusBadRequest)
		return
	}

	// 构造HTTP请求字符串
	requestLine := assembleHTTPRequest("POST", host, path, "", requestData)

	// 发送请求数据
	_, err = conn.Write([]byte(requestLine))
	if err != nil {
		ctx.Error("发送请求失败", fasthttp.StatusInternalServerError)
		return
	}

	// 从连接中读取完整的 HTTP 响应
	httpResponse, err := readFullResponse(conn)
	if err != nil {
		ctx.Error("读取或解析HTTP响应失败", fasthttp.StatusInternalServerError)
		return
	}

	// 获取响应体
	body, err := getResponseBody(httpResponse)
	if err != nil {
		ctx.Error("读取响应体失败", fasthttp.StatusInternalServerError)
		return
	}

	// 返回响应给客户端
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(body)

	log.Printf("请求已发送到 %s，收到响应大小 %d 字节", targetURL, len(body))
}

// parseURLComponents takes a URL string and returns the IP, port, path, and parameters.
func parseURLStr(urlStr string) (string, string, string, error) {
	// Parse the URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to parse URL: %w", err)
	}

	// Extract IP and port
	host := u.Hostname()
	port := u.Port()

	// Extract path
	path := u.Path

	// Extract query parameters
	params := u.RawQuery

	return host + ":" + port, path, params, nil
	// Format and return the result
	//return fmt.Sprintf("IP: %s, Port: %s, Path: %s, Params: %s", host, port, path, params), nil
}

// readFullResponse 从连接中读取完整的 HTTP 响应，并返回一个 HTTP 响应结构体。
func readFullResponse(conn io.Reader) (*http.Response, error) {
	var buf bytes.Buffer
	tmp := make([]byte, 4096) // 初始缓冲区大小

	for {
		n, err := conn.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}

	// 使用 bufio.NewReader 包装缓冲区以创建一个 reader
	reader := bufio.NewReader(&buf)

	// 使用 http.ReadResponse 解析 HTTP 响应
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// getResponseBody 从 HTTP 响应中获取响应体。
func getResponseBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close() // 确保关闭 Body

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())
	switch path {
	case "/open":
		handleOpenConnection(ctx)
	case "/send":
		handleSendRequest(ctx)
	default:
		ctx.Error("Not Found", fasthttp.StatusNotFound)
	}
}

func main() {
	// 设置HTTP服务器
	port := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		port = ":" + p
	}

	log.Printf("服务器启动在端口 %s", port)
	if err := fasthttp.ListenAndServe(port, requestHandler); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}
