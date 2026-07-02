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
	caCertPath = "/Users/hanjian/work/github/cert/ca.crt"      // CA 根证书，用于验证客户端证书
	serverCert = "/Users/hanjian/work/github/cert/app_san.crt" // 服务端证书（含 SAN 扩展）
	serverKey  = "/Users/hanjian/work/github/cert/app.key"     // 服务端私钥
	listenAddr = ":8443"
)

func main() {
	// 1. 注册路由
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})

	// 2. 读取 CA 根证书，用于验证客户端证书
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		log.Fatalf("读取 CA 证书失败: %v", err)
	}

	// 3. 创建 CertPool 并添加 CA 证书
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		log.Fatal("解析 CA 证书失败")
	}

	// 4. 加载服务端证书和私钥
	cert, err := tls.LoadX509KeyPair(serverCert, serverKey)
	if err != nil {
		log.Fatalf("加载服务端证书失败: %v", err)
	}

	// 5. 配置 TLS（开启 mTLS：要求并验证客户端证书）
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,                     // 信任的客户端 CA
		ClientAuth:   tls.RequireAndVerifyClientCert, // 强制双向认证
		MinVersion:   tls.VersionTLS12,               // 最低 TLS 1.2
	}

	// 6. 创建 HTTPS 服务器
	server := &http.Server{
		Addr:      listenAddr,
		TLSConfig: tlsConfig,
	}

	log.Printf("🚀 HTTPS 服务启动，监听 %s\n", listenAddr)
	// 证书已在 TLSConfig.Certificates 中配置，传空字符串即可
	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
