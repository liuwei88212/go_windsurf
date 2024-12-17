package httpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// helloHandler 简单的处理函数，返回一个欢迎消息
func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, you've reached %s!", r.URL.Path)
}

// getHandler 处理GET请求，返回一个简单的JSON响应
func getHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "只支持GET请求", http.StatusMethodNotAllowed)
		return
	}

	params := r.URL.Query()
	result := Result{Code: 0, Msg: "这是一个GET请求的响应", Data: params}
	response, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "无法生成响应", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

// Result 结构体用于POST请求的响应
type Result struct {
	Code     int         `json:"code"`
	Msg      string      `json:"msg"`
	Data     interface{} `json:"data"`
	Duration float64     `json:"duration"` // 耗时（毫秒）
}

// postHandler 处理POST请求，返回请求体
func postHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST请求", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "读取请求体失败", http.StatusBadRequest)
		return
	}

	duration := time.Since(startTime)
	result := Result{
		Code:     0,
		Msg:      "",
		Data:     string(body),
		Duration: float64(duration.Nanoseconds()) / 1e6, // 转换为毫秒
	}

	response, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "无法生成响应", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

// responseWriter 包装 http.ResponseWriter 以捕获状态码和响应内容
type responseWriter struct {
	http.ResponseWriter
	status int
	body   []byte
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body = b
	return rw.ResponseWriter.Write(b)
}

// loggingMiddleware 记录所有HTTP请求的中间件
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// 读取请求体
		var reqBody string
		if r.Body != nil {
			bodyBytes, _ := ioutil.ReadAll(r.Body)
			reqBody = string(bodyBytes)
			// 重新设置请求体，因为ReadAll会清空r.Body
			r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 包装ResponseWriter以捕获响应
		rw := &responseWriter{
			ResponseWriter: w,
			status:        http.StatusOK,
		}

		// 处理请求
		next(rw, r)

		// 获取查询参数
		queryParams := r.URL.Query().Encode()
		if queryParams == "" {
			queryParams = "-"
		}

		// 如果请求体为空，显示为-
		if reqBody == "" {
			reqBody = "-"
		}

		// 如果响应体为空，显示为-
		respBody := string(rw.body)
		if respBody == "" {
			respBody = "-"
		}

		// 计算完整处理时间（包括响应写入）
		duration := time.Since(startTime)
		durationMs := float64(duration.Nanoseconds()) / 1e6 // 转换为毫秒

		// 打印请求信息
		log.Printf("\n请求信息:\n"+
			"路径: %s\n"+
			"方法: %s\n"+
			"参数: %s\n"+
			"请求体: %s\n"+
			"响应状态: %d\n"+
			"响应体: %s\n"+
			"处理时间: %.2fms\n",
			r.URL.Path,
			r.Method,
			queryParams,
			reqBody,
			rw.status,
			respBody,
			durationMs)
	}
}

// StartServer 启动HTTP服务器
func StartServer() {
	// 设置路由
	http.HandleFunc("/", loggingMiddleware(helloHandler))
	http.HandleFunc("/get", loggingMiddleware(getHandler))
	http.HandleFunc("/post", loggingMiddleware(postHandler))

	// 启动HTTP服务器
	port := ":8081"
	log.Printf("HTTP服务器启动在端口 %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("无法启动HTTP服务器: %v", err)
	}
}
