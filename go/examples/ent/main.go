package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/daymenu/gostudy/examples/ent/biz"
	"github.com/daymenu/gostudy/examples/ent/ent"

	// runtime
	_ "github.com/daymenu/gostudy/examples/ent/ent/runtime"

	// 初始化数据库
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	client, err := ent.Open("mysql", "root:123456@tcp(127.0.0.1:3306)/ent?parseTime=True")
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}
	defer client.Close()
	client = client.Debug()
	// Run the auto migration tool.
	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}
	timeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// fmt.Println(biz.QueryCardAmount(timeCtx, client))
	// fmt.Println(biz.CreateGroup(timeCtx, client))
	// fmt.Println(biz.QueryGroup(timeCtx, client))
	fmt.Println(biz.CreateUser(timeCtx, client))
}
