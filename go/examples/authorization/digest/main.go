package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var (
	// 用户数据库：username -> password
	users = map[string]string{
		"admin": "Admin@2021",
		"user":  "User@2021",
		"guest": "Guest@2021",
	}

	// 服务器密钥（用于生成 nonce）
	serverSecret = "my-secret-key-2024"

	// nonce 过期时间（秒）
	nonceExpiration = 300
)

// MD5 哈希
func md5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// 生成 nonce
func generateNonce() string {
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano())
	data := fmt.Sprintf("%s:%s", serverSecret, timestamp)
	return md5Hash(data)
}

// 验证 nonce 是否有效
func validateNonce(nonce string) bool {
	// 简单实现：只检查格式，实际应该检查时间戳和签名
	return len(nonce) == 32 && nonce != ""
}

// 计算期望的 response 值
// HA1 = MD5(username:realm:password)
// HA2 = MD5(method:uri)
// response = MD5(HA1:nonce:HA2)
func calculateExpectedResponse(username, realm, password, nonce, method, uri string) string {
	ha1 := md5Hash(fmt.Sprintf("%s:%s:%s", username, realm, password))
	ha2 := md5Hash(fmt.Sprintf("%s:%s", method, uri))
	response := md5Hash(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))
	return response
}

// 解析 Authorization 头中的 digest 参数
func parseDigestAuth(authHeader string) map[string]string {
	params := make(map[string]string)

	// 移除 "Digest " 前缀
	if !strings.HasPrefix(authHeader, "Digest ") {
		return params
	}

	authData := strings.TrimPrefix(authHeader, "Digest ")

	fmt.Printf("[Debug] Auth data: %s\n", authData)

	// 解析 key="value" 对 - 处理更复杂的格式
	currentKey := ""
	currentValue := ""
	inQuotes := false

	for i := 0; i < len(authData); i++ {
		char := authData[i]

		if char == ' ' && currentKey == "" {
			continue
		}

		if char == '=' && !inQuotes {
			currentKey = strings.TrimSpace(authData[i-len(currentKey) : i])
		} else if char == '"' {
			if inQuotes {
				// End of quoted value
				params[currentKey] = currentValue
				currentKey = ""
				currentValue = ""
				inQuotes = false
			} else {
				inQuotes = true
			}
		} else if char == ',' && !inQuotes {
			// Unquoted value (rare but possible)
			if currentKey != "" && currentValue != "" {
				params[currentKey] = strings.TrimSpace(currentValue)
				currentKey = ""
				currentValue = ""
			}
		} else if inQuotes || (currentKey != "" && char != ' ') {
			currentValue += string(char)
		}
	}

	// Handle last parameter
	if currentKey != "" && currentValue != "" {
		params[currentKey] = strings.TrimSpace(currentValue)
	}

	fmt.Printf("[Debug] Parsed params: %+v\n", params)
	return params
}

// Digest 认证中间件
func digestAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	realm := "Restricted Area"

	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			// 没有认证信息，发送质询
			sendDigestChallenge(w, realm)
			return
		}

		// 解析认证参数
		params := parseDigestAuth(authHeader)

		username := params["username"]
		nonce := params["nonce"]
		response := params["response"]
		uri := params["uri"]

		// 验证必要参数
		if username == "" || nonce == "" || response == "" || uri == "" {
			sendDigestChallenge(w, realm)
			return
		}

		// 获取用户密码
		password, exists := users[username]
		if !exists {
			sendDigestChallenge(w, realm)
			return
		}

		// 验证 nonce
		if !validateNonce(nonce) {
			sendDigestChallenge(w, realm)
			return
		}

		// 计算期望的 response
		expectedResponse := calculateExpectedResponse(
			username, realm, password,
			nonce, r.Method, uri,
		)

		// 验证 response
		if response != expectedResponse {
			fmt.Printf("[Debug] Response mismatch: got=%s, expected=%s\n", response, expectedResponse)
			sendDigestChallenge(w, realm)
			return
		}

		// 认证成功
		fmt.Printf("[Auth] User '%s' authenticated successfully\n", username)
		next(w, r)
	}
}

// 发送  质询
func sendDigestChallenge(w http.ResponseWriter, realm string) {
	nonce := generateNonce()

	// WWW-Authenticate: Digest realm="...", nonce="...", qop="auth"
	challenge := fmt.Sprintf(
		`Digest realm="%s", nonce="%s", qop="auth"`,
		realm, nonce,
	)

	w.Header().Set("WWW-Authenticate", challenge)
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintf(w, `{"error": "Unauthorized", "message": "Digest authentication required"}`)
}

// 处理 API 请求
func handleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "success", "message": "Access granted to protected resource"}`)
}

func main() {
	http.HandleFunc("/api/", digestAuthMiddleware(handleAPI))

	fmt.Println("Server starting on :8080")
	fmt.Println("\nDigest Authentication Example")
	fmt.Println("================================")
	fmt.Println("\nTest with curl:")
	fmt.Println(`  curl -u admin:Admin@2021 --digest http://127.0.0.1:8080/api/resource`)
	fmt.Println("\nOr manually:")
	fmt.Println("1. First request (will get 401 with WWW-Authenticate header)")
	fmt.Println("   curl -v http://127.0.0.1:8080/api/resource")
	fmt.Println("\n2. Calculate response and send with Authorization header")

	http.ListenAndServe(":8080", nil)
}
