package requestutil

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Get请求
func HttpGet(url string) (res []byte, err error) {

	if url == "" {
		log.Println("[REQUEST UTIL] Error, url is empty ", err)
		return
	}

	// 初始化请求
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("[REQUEST UTIL] Error, http GET ", err)
		return
	}

	// 添加请求头, 发送请求
	request.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Println("[REQUEST UTIL] Error, http.DefaultClient.Do, ", err)
		return
	}

	// 读取响应数据
	// respBody, err := ioutil.ReadAll(resp.Body)
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("[REQUEST UTIL] Error, io.ReadAll, ", err)
		return
	}
	// 手动释放, 不交给defer
	resp.Body.Close()

	res = respBody
	log.Println("[REQUEST UTIL] Response data: ", string(respBody))
	return
}

// Post请求
func HttpPost(url string, data string) (res []byte, err error) {

	if url == "" {
		log.Println("[REQUEST UTIL] Error, url is empty ", err)
		return
	}

	// 初始化请求
	request, err := http.NewRequest("POST", url, strings.NewReader(data))
	if err != nil {
		log.Println("[REQUEST UTIL] Error, http post ", err)
		return
	}

	// 添加请求头, 发送请求
	request.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Println("[REQUEST UTIL] Error, http.DefaultClient.Do, ", err)
		return
	}

	// 读取响应数据
	// respBody, err := ioutil.ReadAll(resp.Body)
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("[REQUEST UTIL] Error, io.ReadAll, ", err)
		return
	}
	// 手动释放, 不交给defer
	resp.Body.Close()

	res = respBody
	log.Println("[REQUEST UTIL] Response data: ", string(respBody))
	return
}

// PostForm
func PostForm(url string, data url.Values) (res []byte, err error) {
	resp, err := http.PostForm(url, data)

	if err != nil {
		log.Println("[REQUEST UTIL] PostForm Error, http.DefaultClient.Do, ", err)
		return
	}
	defer resp.Body.Close()

	res, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Println("[REQUEST UTIL] PostForm Error, io.ReadAll, ", err)
		return
	}

	fmt.Println(string(res))
	return
}

// PostMultipartForm 发送 multipart/form-data 请求。
func PostMultipartForm(url string, data map[string]string) (res []byte, err error) {
	return PostMultipartFormWithTimeout(url, data, 10*time.Second)
}

// PostMultipartFormWithTimeout 发送带超时时间的 multipart/form-data 请求。
func PostMultipartFormWithTimeout(url string, data map[string]string, timeout time.Duration) (res []byte, err error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for k, v := range data {
		if err = writer.WriteField(k, v); err != nil {
			log.Println("[REQUEST UTIL] PostMultipartForm WriteField Error, ", err)
			return
		}
	}
	if err = writer.Close(); err != nil {
		log.Println("[REQUEST UTIL] PostMultipartForm Close Error, ", err)
		return
	}

	request, err := http.NewRequest("POST", url, &body)
	if err != nil {
		log.Println("[REQUEST UTIL] PostMultipartForm NewRequest Error, ", err)
		return
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Mozilla/5.0")
	request.Close = true

	transport := &http.Transport{
		// 短信接口固定直连，不读取 HTTP_PROXY、HTTPS_PROXY、NO_PROXY 等代理环境变量。
		Proxy:                 nil,
		DisableKeepAlives:     true,
		ForceAttemptHTTP2:     false,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: time.Second,
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
	resp, err := client.Do(request)
	if err != nil {
		log.Println("[REQUEST UTIL] PostMultipartForm Do Error, ", err)
		return
	}
	defer resp.Body.Close()
	defer transport.CloseIdleConnections()

	res, err = io.ReadAll(resp.Body)

	if err != nil {
		log.Println("[REQUEST UTIL] PostMultipartForm ReadAll Error, ", err)
		return
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		err = fmt.Errorf("http status %s: %s", resp.Status, string(res))
		log.Println("[REQUEST UTIL] PostMultipartForm Status Error, ", err)
	}
	return
}

// 添加证书的代码
func GetHttps(url, caCertPath, certFile, keyFile string) ([]byte, error) {

	// 创建证书池及各类对象
	var pool *x509.CertPool // 我们要把一部分证书存到这个池中
	var client *http.Client
	var resp *http.Response
	var body []byte
	var err error

	var caCrt []byte // 根证书
	// caCrt, err = ioutil.ReadFile(caCertPath)
	caCrt, err = os.ReadFile(caCertPath)
	pool = x509.NewCertPool()
	if err != nil {
		log.Println("[REQUEST UTIL] GetHttps CertPool Error:", err)
		return nil, err
	}
	pool.AppendCertsFromPEM(caCrt)

	var cliCrt tls.Certificate // 具体的证书加载对象
	cliCrt, err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Println("[REQUEST UTIL] GetHttps Load KeyPair Error:", err)
		return nil, err
	}

	// 把上面的准备内容传入 client
	client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      pool,
				Certificates: []tls.Certificate{cliCrt},
			},
		},
	}

	// Get 请求
	resp, err = client.Get(url)
	if err != nil {
		log.Println("[REQUEST UTIL] GetHttps Request Error:", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Println("[REQUEST UTIL] GetHttps Read Response Error:", err)
		return nil, err
	}
	defer client.CloseIdleConnections()
	return body, nil
}

// 在 Header 中添加 token 的 HTTPS 请求
func GetHttpsSkip(url, token string) ([]byte, error) {

	// 创建各类对象
	var client *http.Client
	var request *http.Request
	var resp *http.Response
	var body []byte
	var err error

	/** `这里请注意，使用 InsecureSkipVerify: true 来跳过证书验证` */
	client = &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}}

	// 获取 request请求
	request, err = http.NewRequest("GET", url, nil)

	if err != nil {
		log.Println("[REQUEST UTIL] GetHttpsSkip Request Error:", err)
		return nil, nil
	}

	// 加入 token
	request.Header.Add("Authorization", token)
	resp, err = client.Do(request)
	if err != nil {
		log.Println("[REQUEST UTIL] GetHttpsSkip Response Error:", err)
		return nil, nil
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	defer client.CloseIdleConnections()

	return body, nil
}
