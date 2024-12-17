package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// HostConnections 存储每个host的多个连接
type HostConnections struct {
	Connections []net.Conn
	mutex       sync.RWMutex
	lastUsed    int // 用于轮询方式使用连接
}

// 用于存储TCP连接的map
var (
	connections = make(map[string]*HostConnections)
	connMutex   sync.RWMutex
)

// ConnectRequest TCP连接请求的结构体
type ConnectRequest struct {
	TargetURL string `json:"targetUrl"`
	Count     int    `json:"count"` // 要创建的连接数量
}

// SendRequest HTTP请求的结构体
type SendRequest struct {
	TargetURL  string            `json:"targetUrl"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       interface{}       `json:"body,omitempty"`
	Concurrent int               `json:"concurrent"` // 并发发送的数量
}

// SendResponse 发送响应的结构体
type SendResponse struct {
	StatusCode int                 `json:"statusCode"`
	Headers    map[string][]string `json:"headers"`
	Body       string              `json:"body"`
	TimeTaken  string              `json:"timeTaken"`
}

// getNextConnection 以轮询方式获取下一个可用连接
func (hc *HostConnections) getNextConnection() net.Conn {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	hc.lastUsed = (hc.lastUsed + 1) % len(hc.Connections)
	return hc.Connections[hc.lastUsed]
}

// TCPConnectionHandler 处理TCP连接的建立
func TCPConnectionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只允许POST方法", http.StatusMethodNotAllowed)
		return
	}

	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "解析JSON请求失败: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.TargetURL == "" {
		http.Error(w, "targetUrl不能为空", http.StatusBadRequest)
		return
	}

	if req.Count <= 0 {
		req.Count = 1 // 默认创建一个连接
	}

	parsedURL, err := url.Parse(req.TargetURL)
	if err != nil {
		http.Error(w, "URL格式无效", http.StatusBadRequest)
		return
	}

	host := parsedURL.Host
	if !strings.Contains(host, ":") {
		if parsedURL.Scheme == "https" {
			host += ":443"
		} else {
			host += ":80"
		}
	}

	// 创建多个连接
	hostConns := &HostConnections{
		Connections: make([]net.Conn, 0, req.Count),
	}

	for i := 0; i < req.Count; i++ {
		conn, err := net.Dial("tcp", host)
		if err != nil {
			// 如果创建连接失败，关闭已创建的连接
			for _, c := range hostConns.Connections {
				c.Close()
			}
			http.Error(w, fmt.Sprintf("创建第%d个连接失败: %v", i+1, err), http.StatusInternalServerError)
			return
		}
		hostConns.Connections = append(hostConns.Connections, conn)
	}

	// 存储连接
	connMutex.Lock()
	if existing, exists := connections[host]; exists {
		// 如果已存在连接，关闭它们
		for _, conn := range existing.Connections {
			conn.Close()
		}
	}
	connections[host] = hostConns
	connMutex.Unlock()

	response := map[string]interface{}{
		"message": fmt.Sprintf("已建立 %d 个TCP连接到 %s", req.Count, host),
		"status":  "success",
		"count":   req.Count,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("已建立 %d 个TCP连接到 %s", req.Count, host)
}

// sendSingleRequest 发送单个请求并返回响应
func sendSingleRequest(conn net.Conn, httpRequest string) (*SendResponse, error) {
	startTime := time.Now()

	_, err := conn.Write([]byte(httpRequest))
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}

	response, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %v", err)
	}

	return &SendResponse{
		StatusCode: response.StatusCode,
		Headers:    response.Header,
		Body:       string(responseBody),
		TimeTaken:  time.Since(startTime).String(),
	}, nil
}

// HTTPRequestHandler 处理HTTP请求的发送
func HTTPRequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只允许POST方法", http.StatusMethodNotAllowed)
		return
	}

	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "解析JSON请求失败: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.TargetURL == "" {
		http.Error(w, "targetUrl不能为空", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.Parse(req.TargetURL)
	if err != nil {
		http.Error(w, "URL格式无效", http.StatusBadRequest)
		return
	}

	host := parsedURL.Host
	port := parsedURL.Port()
	if !strings.Contains(host, ":") {
		if parsedURL.Scheme == "https" {
			host += ":443"
		} else {
			host += ":" + port
		}
	}

	connMutex.RLock()
	hostConns, exists := connections[host]
	connMutex.RUnlock()

	if !exists || len(hostConns.Connections) == 0 {
		http.Error(w, "未找到已建立的连接", http.StatusBadRequest)
		return
	}

	bodyBytes, err := json.Marshal(req.Body)
	if err != nil {
		http.Error(w, "序列化请求体失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	method := req.Method
	if method == "" {
		method = "POST"
	}

	// 构造HTTP请求字符串
	httpRequest := fmt.Sprintf("%s %s HTTP/1.1\r\n", method, parsedURL.Path)
	httpRequest += fmt.Sprintf("Host: %s\r\n", parsedURL.Host)
	httpRequest += "Content-Type: application/json\r\n"
	httpRequest += fmt.Sprintf("Content-Length: %d\r\n", len(bodyBytes))

	for key, value := range req.Headers {
		httpRequest += fmt.Sprintf("%s: %s\r\n", key, value)
	}
	httpRequest += "\r\n"
	httpRequest += string(bodyBytes)

	// 设置并发数
	concurrent := req.Concurrent
	if concurrent <= 0 {
		concurrent = 1
	}
	if concurrent > len(hostConns.Connections) {
		concurrent = len(hostConns.Connections)
	}

	// 并发发送请求
	responses := make([]*SendResponse, concurrent)
	var wg sync.WaitGroup
	errChan := make(chan error, concurrent)

	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			conn := hostConns.getNextConnection()
			resp, err := sendSingleRequest(conn, httpRequest)
			if err != nil {
				errChan <- err
				return
			}
			responses[index] = resp
		}(i)
	}

	wg.Wait()
	close(errChan)

	// 检查是否有错误
	if len(errChan) > 0 {
		var errMsgs []string
		for err := range errChan {
			errMsgs = append(errMsgs, err.Error())
		}
		http.Error(w, fmt.Sprintf("部分请求失败: %s", strings.Join(errMsgs, "; ")), http.StatusInternalServerError)
		return
	}

	// 返回所有响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"responses": responses,
		"count":     concurrent,
	})
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	http.HandleFunc("/connect", TCPConnectionHandler)
	http.HandleFunc("/send", HTTPRequestHandler)

	port := ":8080"
	log.Printf("服务器启动在端口%s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
