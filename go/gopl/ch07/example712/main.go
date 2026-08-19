package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"maps"
	"net/http"
	"slices"
)

// **练习 7.12：** 修改/list的handler让它把输出打印成一个HTML的表格而不是文本。
// html/template包（§4.6）可能会对你有帮助。

//go:embed list.html
var listTmplText string

// 模板只在启动时解析一次：解析失败属于程序错误，应该立刻 panic 而不是等到请求到来。
var listTmpl = template.Must(template.New("list").Parse(listTmplText))

type dollars float32

func (d dollars) String() string { return fmt.Sprintf("$%.2f", d) }

type database map[string]dollars

// item 是模板的行数据；用切片而不是 map，保证表格顺序稳定。
type item struct {
	Name  string
	Price dollars
}

func main() {
	db := database{"shoes": 50, "socks": 5}
	http.HandleFunc("/list", db.List)
	http.HandleFunc("/price", db.Price)
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

func (db database) List(w http.ResponseWriter, req *http.Request) {
	items := make([]item, 0, len(db))
	for _, name := range slices.Sorted(maps.Keys(db)) {
		items = append(items, item{Name: name, Price: db[name]})
	}

	data := struct {
		Items []item
	}{Items: items}

	// 先渲染到缓冲区再一次性写出，这样渲染中途出错时响应头还没发出去，仍能返回 500。
	var buf bytes.Buffer
	if err := listTmpl.Execute(&buf, data); err != nil {
		log.Printf("render list: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError) // 500
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

func (db database) Price(w http.ResponseWriter, req *http.Request) {
	name := req.URL.Query().Get("item")
	price, ok := db[name]
	if !ok {
		w.WriteHeader(http.StatusNotFound) // 404
		fmt.Fprintf(w, "no such item: %q\n", name)
		return
	}
	fmt.Fprintf(w, "%s\n", price)
}
