package main

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"net/http"
	"os"
)

// 证书文件路径（mTLS 双向认证场景）
const (
	caCertPath = "/Users/hanjian/work/github/cert/ca.crt"      // CA 根证书，用于验证服务端证书
	clientCert = "/Users/hanjian/work/github/cert/app_san.crt" // 客户端证书（向服务端证明身份）
	clientKey  = "/Users/hanjian/work/github/cert/app.key"     // 客户端私钥
	serverURL  = "https://localhost:8443"
)

func main() {
	// 1. 读取 CA 根证书
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		log.Fatalf("读取 CA 证书失败: %v", err)
	}

	// 2. 创建 CertPool 并添加 CA 证书，使客户端信任该 CA 签发的服务端证书
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		log.Fatal("解析 CA 证书失败")
	}

	// 3. 加载客户端证书和私钥（mTLS：向服务端证明客户端身份）
	cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
	if err != nil {
		log.Fatalf("加载客户端证书失败: %v", err)
	}

	// 4. 配置 TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert}, // 客户端证书，用于 mTLS 认证
		RootCAs:      caCertPool,               // 信任的 CA，用于验证服务端证书
		MinVersion:   tls.VersionTLS12,         // 最低 TLS 1.2
	}

	// 5. 创建带有自定义 TLS 配置的 HTTP 客户端
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// 6. 发起请求
	resp, err := client.Get(serverURL)
	if err != nil {
		log.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	log.Println("✅ 请求成功，状态码:", resp.Status)
}
