package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// 练习 4.12： 流行的web漫画服务xkcd也提供了JSON接口。例如，一个 https://xkcd.com/571/info.0.json 请求将返回一个很多人喜爱的571编号的详细描述。
// 下载每个链接（只下载一次）然后创建一个离线索引。编写一个xkcd工具，使用这些离线索引，打印和命令行输入的检索词相匹配的漫画的URL。

const XKCDURL = "https://xkcd.com/%d/info.0.json"

func main() {
	indexXKCDs()
}

func indexXKCDs() error {
	i := 1
	for {
		log.Printf("Fetching XKCD %d", i)
		if isCached(i) {
			i++
			continue
		}

		data, err := fetchXKCD(i)
		if err != nil {
			if os.IsNotExist(err) {
				break
			}
			return err
		}
		err = indexXKCD(i, data)
		if err != nil {
			return err
		}
		i++
	}
	return nil
}

func fetchXKCD(num int) (string, error) {
	url := getURL(num)
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func indexXKCD(num int, data string) error {
	dir := getCacheDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
		if err != nil {
			return err
		}
	}
	filename := fmt.Sprintf("%s/%d.json", dir, num)
	return os.WriteFile(filename, []byte(data), 0644)
}

func isCached(num int) bool {
	filename := fmt.Sprintf("%s/%d.json", getCacheDir(), num)
	_, err := os.Stat(filename)
	return err == nil
}

func getCacheDir() string {
	return "./xkcd_cache"
}

func getXKCD(num int) (string, error) {
	filename := fmt.Sprintf("%s/%d.json", getCacheDir(), num)
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func getURL(num int) string {
	return fmt.Sprintf(XKCDURL, num)
}
