package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

// 练习 4.12： 流行的web漫画服务xkcd也提供了JSON接口。例如，一个 https://xkcd.com/571/info.0.json 请求将返回一个很多人喜爱的571编号的详细描述。
// 下载每个链接（只下载一次）然后创建一个离线索引。编写一个xkcd工具，使用这些离线索引，打印和命令行输入的检索词相匹配的漫画的URL。

const XKCDURL = "https://xkcd.com/%d/info.0.json"

// errXKCDNotFound 表示请求的漫画编号不存在（HTTP 404），用于终止索引循环。
var errXKCDNotFound = errors.New("xkcd: comic not found")

type XKCD struct {
	Month      string `json:"month"`
	Num        int    `json:"num"`
	Link       string `json:"link"`
	Year       string `json:"year"`
	News       string `json:"news"`
	SafeTitle  string `json:"safe_title"`
	Transcript string `json:"transcript"`
	Alt        string `json:"alt"`
	Img        string `json:"img"`
	Title      string `json:"title"`
	Day        string `json:"day"`
}

func main() {
	indexXKCDs()
	log.Println("Enter a search term:")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		searchTerm := strings.TrimSpace(scanner.Text())
		if searchTerm == "exit" {
			break
		}
		results, err := searchXKCDs(searchTerm)
		if err != nil {
			log.Fatalf("Search failed: %v", err)
		}
		for _, xkcd := range results {
			fmt.Printf("Title: %s\nURL: %s\n\n", xkcd.Title, xkcd.Img)
		}

		log.Println("Enter a search term:")
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading input: %v", err)
	}
}

func searchXKCDs(searchTerm string) ([]XKCD, error) {
	cached, err := getCachedXKCDs()
	if err != nil {
		return nil, err
	}
	var results []XKCD
	for _, xkcd := range cached {
		if strings.Contains(xkcd.Title, searchTerm) || strings.Contains(xkcd.Transcript, searchTerm) {
			results = append(results, xkcd)
		}
	}
	return results, nil
}

func getCachedXKCDs() ([]XKCD, error) {
	dir := getCacheDir()
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var cached []XKCD
	for _, file := range files {
		if !file.IsDir() {
			filePath := path.Join(dir, file.Name())
			xkcd, err := getXKCDFromFile(filePath)
			if err != nil {
				log.Printf("Error reading cached XKCD %s: %v", file.Name(), err)
				continue
			}
			cached = append(cached, xkcd)
		}
	}
	return cached, nil
}

func getXKCDFromFile(filename string) (XKCD, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return XKCD{}, err
	}
	var xkcd XKCD
	err = json.Unmarshal(data, &xkcd)
	if err != nil {
		return XKCD{}, err
	}
	return xkcd, nil
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
			if errors.Is(err, errXKCDNotFound) {
				break
			}
			log.Printf("Fetching XKCD %d failed: %v", i, err)
			continue
		}
		err = indexXKCD(i, data)
		if err != nil {
			log.Printf("Indexing XKCD %d failed: %v", i, err)
			continue
		}
		i++
	}
	return nil
}

func fetchXKCD(num int) (string, error) {
	xkcdURL := getURL(num)
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(xkcdURL)
	if err != nil {
		return "", errXKCDNotFound
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound && num != 404 {
		return "", errXKCDNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code: %d", resp.StatusCode)
	}
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
	filename := getFilePath(num)
	return os.WriteFile(filename, []byte(data), 0644)
}

func isCached(num int) bool {
	filename := getFilePath(num)
	_, err := os.Stat(filename)
	return err == nil
}

func getFilePath(num int) string {
	return fmt.Sprintf("%s/%d.json", getCacheDir(), num)
}

func getCacheDir() string {
	return "./xkcd_cache"
}

func getXKCD(num int) (string, error) {
	filename := getFilePath(num)
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func getURL(num int) string {
	return fmt.Sprintf(XKCDURL, num)
}
