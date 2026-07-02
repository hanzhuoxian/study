package main

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	// 1. 创建 etcd 客户端连接
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal("连接 etcd 失败:", err)
	}
	defer cli.Close()
	fmt.Println("成功连接到 etcd")

	// 2. 写入数据（Put）
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err = cli.Put(ctx, "/config/db/host", "localhost")
	cancel()
	if err != nil {
		log.Fatal("写入失败:", err)
	}
	fmt.Println("写入成功: /config/db/host = localhost")

	// 3. 读取数据（Get）
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	resp, err := cli.Get(ctx, "/config/db/host")
	cancel()
	if err != nil {
		log.Fatal("读取失败:", err)
	}
	for _, kv := range resp.Kvs {
		fmt.Printf("读取结果: %s = %s\n", kv.Key, kv.Value)
	}

	// 4. 监听数据变化（Watch）
	fmt.Println("开始监听 /config/db/host 的变化...")
	go func() {
		rch := cli.Watch(context.Background(), "/config/db/host")
		for wresp := range rch {
			for _, ev := range wresp.Events {
				fmt.Printf("事件类型: %s, 键: %s, 值: %s\n",
					ev.Type, ev.Kv.Key, ev.Kv.Value)
			}
		}
	}()

	// 5. 使用租约（Lease）—— 带 TTL 的 key
	// 创建一个 10 秒的租约
	leaseResp, err := cli.Grant(context.Background(), 10)
	if err != nil {
		log.Fatal("创建租约失败:", err)
	}
	fmt.Printf("创建租约成功, LeaseID: %d\n", leaseResp.ID)

	// 把 key 绑定到租约上，10 秒后自动删除
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	_, err = cli.Put(ctx, "/temp/key", "will-expire", clientv3.WithLease(leaseResp.ID))
	cancel()
	if err != nil {
		log.Fatal("写入带租约的 key 失败:", err)
	}
	fmt.Println("写入带租约的 key 成功，10 秒后自动过期")

	// 6. 验证租约 key 是否存在
	time.Sleep(2 * time.Second)
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	resp, err = cli.Get(ctx, "/temp/key")
	cancel()
	if err != nil {
		log.Fatal("读取失败:", err)
	}
	if len(resp.Kvs) == 0 {
		fmt.Println("/temp/key 已过期删除")
	} else {
		fmt.Printf("/temp/key 仍然存在: %s\n", resp.Kvs[0].Value)
	}

	// 7. 删除数据（Delete）
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	_, err = cli.Delete(ctx, "/config/db/host")
	cancel()
	if err != nil {
		log.Fatal("删除失败:", err)
	}
	fmt.Println("删除成功: /config/db/host")

	// 保持程序运行，以便观察 Watch 效果
	time.Sleep(5 * time.Second)
	fmt.Println("程序结束")
}
