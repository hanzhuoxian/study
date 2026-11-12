package main

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

var sg singleflight.Group

func GetData(key string) (interface{}, error) {
	v, err, _ := sg.Do(key, func() (interface{}, error) {
		// 这里是真正的耗时操作，例如查询数据库
		time.Sleep(1 * time.Second)
		fmt.Println("get data")
		return "hello world", nil
	})
	return v, err
}

func main() {
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Go(func() {
			v, err := GetData("key")
			if err != nil {
				panic(err)
			}
			fmt.Println(v.(string))
		})
	}
	wg.Wait()
}
