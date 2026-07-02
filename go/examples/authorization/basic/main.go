package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

var (
	users = map[string]string{
		"admin": "Admin@2021",
		"user":  "User@2021",
		"guest": "Guest@2021",
	}
)

func main() {
	http.HandleFunc("/api/", authMiddleware(handleAPI))

	fmt.Println("Server starting on :8080")
	fmt.Println("\nTest examples:")
	fmt.Println(`  basic=$(echo -n 'admin:Admin@2021' | base64)`)
	fmt.Println(`  curl -H "Authorization: Basic ${basic}" http://127.0.0.1:8080/api/resource`)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Print("err %w", err)
	}
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Basic" {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		payload, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			http.Error(w, "Invalid base64 encoding", http.StatusUnauthorized)
			return
		}

		pair := strings.SplitN(string(payload), ":", 2)
		if len(pair) != 2 {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		username, password := pair[0], pair[1]

		if users[username] != password {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		fmt.Printf("[Auth] User '%s' authenticated successfully\n", username)
		next(w, r)
	}
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "success", "message": "Access granted", "resource": "protected-api"}`)
}
