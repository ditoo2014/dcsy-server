package requestutil

// import (
// 	"bytes"
// 	"context"
// 	"fmt"
// 	"io"
// 	"log"
// 	"net/http"
// 	"net/url"
// 	"strings"
// 	"time"
// )

// // Curl结构体
// type Curl struct {
// 	client  *http.Client      // http client
// 	baseURL string            // base url
// 	headers map[string]string // headers
// }

// // 初始化函数
// func NewCurl(baseURL string, timeout time.Duration) *Curl {
// 	return &Curl{
// 		client: &http.Client{
// 			Timeout: timeout,
// 		},
// 		baseURL: strings.TrimSuffix(baseURL, "/"),
// 		headers: make(map[string]string),
// 	}
// }

// // 设置请求头
// func (c *Curl) SetHeader(key, value string) {
// 	c.headers[key] = value
// }

// // 构建请求
// func (c *Curl) buildRequest(ctx context.Context, method, urlPath string, queryParams map[string]string, body io.Reader) (*http.Request, error) {
// 	// 处理完整的 URL
// 	fullURL := c.baseURL + urlPath
// 	if queryParams != nil {
// 		query := url.Values{}
// 		for key, value := range queryParams {
// 			query.Add(key, value)
// 		}
// 		fullURL += "?" + query.Encode()
// 	}

// 	// 创建请求
// 	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 设置请求头
// 	for key, value := range c.headers {
// 		req.Header.Set(key, value)
// 	}

// 	return req, nil
// }

// // 发送请求
// func (c *Curl) doRequest(req *http.Request) (*http.Response, error) {
// 	resp, err := c.client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return resp, nil
// }

// // get request
// func (c *Curl) Get(ctx context.Context, urlPath string, queryParams map[string]string) (*http.Response, error) {
// 	req, err := c.buildRequest(ctx, http.MethodGet, urlPath, queryParams, nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return c.doRequest(req)
// }

// // post request
// func (c *Curl) Post(ctx context.Context, urlPath string, body []byte) (*http.Response, error) {
// 	req, err := c.buildRequest(ctx, http.MethodPost, urlPath, nil, bytes.NewBuffer(body))
// 	if err != nil {
// 		return nil, err
// 	}
// 	return c.doRequest(req)
// }

// // put request
// func (c *Curl) Put(ctx context.Context, urlPath string, body []byte) (*http.Response, error) {
// 	req, err := c.buildRequest(ctx, http.MethodPut, urlPath, nil, bytes.NewBuffer(body))
// 	if err != nil {
// 		return nil, err
// 	}
// 	return c.doRequest(req)
// }

// // delete request
// func (c *Curl) Delete(ctx context.Context, urlPath string) (*http.Response, error) {
// 	req, err := c.buildRequest(ctx, http.MethodDelete, urlPath, nil, nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return c.doRequest(req)
// }

// // 读取响应体
// func ReadResponseBody(resp *http.Response) ([]byte, error) {
// 	defer func(Body io.ReadCloser) {
// 		err := Body.Close()
// 		if err != nil {
// 			fmt.Printf("close response body failed: %v\n", err)
// 		}
// 	}(resp.Body)
// 	return io.ReadAll(resp.Body)
// }

// // test
// func NormalGet(baseUrl string, appendUrl string, data map[string]string) (body []byte, err error) {
// 	curl := NewCurl(baseUrl, 10*time.Second)

// 	resp, err := curl.Get(context.TODO(), appendUrl, data)
// 	if err != nil {
// 		log.Printf("Failed to send GET request: %v", err)
// 	}

// 	body, err = ReadResponseBody(resp)
// 	if err != nil {
// 		log.Printf("Failed to read response: %v", err)
// 	}

// 	fmt.Println("Response:", string(body))

// 	return
// }
