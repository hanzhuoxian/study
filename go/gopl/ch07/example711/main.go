package main

import (
	"fmt"
	"log"
	"maps"
	"net/http"
	"strconv"
	"sync"
)

// **练习 7.11：** 增加额外的handler让客户端可以创建，读取，更新和删除数据库记录。
// 例如，一个形如 `/update?item=socks&price=6` 的请求会更新库存清单里一个货品的价格并且当这个货品不存在或价格无效时返回一个错误值。
// （注意：这个修改会引入变量同时更新的问题）

type dollars float32

func (d dollars) String() string { return fmt.Sprintf("$%.2f", d) }

type database struct {
	mu    sync.RWMutex
	items map[string]dollars
}

func main() {
	db := &database{items: map[string]dollars{"shoes": 50, "socks": 5}}

	http.HandleFunc("/list", db.List)
	http.HandleFunc("/price", db.Price)
	http.HandleFunc("/create", db.Create)
	http.HandleFunc("/update", db.Update)
	http.HandleFunc("/delete", db.Delete)

	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

func (db *database) List(w http.ResponseWriter, req *http.Request) {
	// 先在锁内取快照，避免慢客户端在写响应期间一直占着锁。
	db.mu.RLock()
	items := maps.Clone(db.items)
	db.mu.RUnlock()

	for item, price := range items {
		fmt.Fprintf(w, "%s: %s\n", item, price)
	}
}

func (db *database) Price(w http.ResponseWriter, req *http.Request) {
	item := req.URL.Query().Get("item")

	db.mu.RLock()
	price, ok := db.items[item]
	db.mu.RUnlock()

	if !ok {
		notFound(w, item)
		return
	}
	fmt.Fprintf(w, "%s\n", price)
}

func (db *database) Create(w http.ResponseWriter, req *http.Request) {
	item, price, ok := parseItemPrice(w, req)
	if !ok {
		return
	}

	// 检查存在与写入必须在同一把写锁内，否则是 check-then-act 竞态。
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.items[item]; exists {
		w.WriteHeader(http.StatusConflict) // 409
		fmt.Fprintf(w, "item already exists: %q\n", item)
		return
	}
	db.items[item] = price
}

func (db *database) Update(w http.ResponseWriter, req *http.Request) {
	item, price, ok := parseItemPrice(w, req)
	if !ok {
		return
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.items[item]; !exists {
		notFound(w, item)
		return
	}
	db.items[item] = price
}

func (db *database) Delete(w http.ResponseWriter, req *http.Request) {
	item := req.URL.Query().Get("item")

	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.items[item]; !exists {
		notFound(w, item)
		return
	}
	delete(db.items, item)
}

// parseItemPrice 解析并校验 item 与 price 参数，校验失败时已写好错误响应并返回 false。
func parseItemPrice(w http.ResponseWriter, req *http.Request) (string, dollars, bool) {
	q := req.URL.Query()

	item := q.Get("item")
	if item == "" {
		w.WriteHeader(http.StatusBadRequest) // 400
		fmt.Fprintln(w, "missing item")
		return "", 0, false
	}

	priceStr := q.Get("price")
	price, err := strconv.ParseFloat(priceStr, 32)
	if err != nil || price < 0 {
		w.WriteHeader(http.StatusBadRequest) // 400
		fmt.Fprintf(w, "invalid price: %q\n", priceStr)
		return "", 0, false
	}

	return item, dollars(price), true
}

func notFound(w http.ResponseWriter, item string) {
	w.WriteHeader(http.StatusNotFound) // 404
	fmt.Fprintf(w, "no such item: %q\n", item)
}
