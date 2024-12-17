package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"net/http"
)

func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	// Log the request URL and headers
	log.Printf("Request URL: %s", req.URL.String())
	for name, values := range req.Header {
		for _, value := range values {
			log.Printf("Header: %s = %s", name, value)
		}
	}

	// Log the request body
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			log.Printf("Body: %s", string(bodyBytes))
			// Restore the io.ReadCloser to its original state
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	}

	log.Printf("Request Send start....")

	// Create a new request based on the incoming request
	outReq, err := http.NewRequest(req.Method, req.URL.String(), req.Body)
	if err != nil {
		http.Error(res, "Server Error", http.StatusInternalServerError)
		return
	}

	// Copy the headers from the incoming request to the outgoing request
	outReq.Header = req.Header

	// Perform the request
	client := &http.Client{}
	resp, err := client.Do(outReq)
	if err != nil {
		http.Error(res, "Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Log the response status and headers
	log.Printf("Response Status: %s", resp.Status)
	for name, values := range resp.Header {
		for _, value := range values {
			log.Printf("Response Header: %s = %s", name, value)
		}
	}

	// Log the response body
	var respBodyBytes []byte

	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			log.Println("Failed to create gzip reader:", err)
			respBodyBytes, _ = io.ReadAll(resp.Body) // Fallback to reading uncompressed body
		} else {
			defer gzipReader.Close() // Ensure gzip reader is always closed
			respBodyBytes, err = io.ReadAll(gzipReader)
			if err != nil {
				log.Println("Failed to read gzip body:", err)
			}
		}
	} else {
		respBodyBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			log.Println("Failed to read response body:", err)
		}
	}

	log.Printf("Response Body: %d", len(respBodyBytes))

	resp.Body = io.NopCloser(bytes.NewBuffer(respBodyBytes))
	log.Printf("Response Body read End...")

	// 使用gzip解析完，需要将gzip去掉，不然postman无法解析
	for key, value := range resp.Header {
		if key != "Content-Encoding" {
			res.Header()[key] = value
		}
	}
	res.WriteHeader(resp.StatusCode)

	// Copy the body from the response to the outgoing response
	io.Copy(res, resp.Body)
}

func main() {
	// Set up the HTTP server
	http.HandleFunc("/", handleRequestAndRedirect)

	// Add a POST test endpoint
	http.HandleFunc("/test-post", func(res http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(res, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		res.WriteHeader(http.StatusOK)
		res.Write([]byte("POST request received"))
	})

	log.Println("Starting proxy server on :8088")
	log.Fatal(http.ListenAndServe(":8088", nil))
}
