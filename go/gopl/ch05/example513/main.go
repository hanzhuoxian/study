package main

// **练习5.13：** 修改crawl，使其能保存发现的页面，必要时，可以创建目录来保存这些页面。只保存来自原始域名下的页面。假设初始页面在golang.org下，就不要保存vimeo.com下的页面。

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/hanzhuoxian/study/go/gopl/ch05/links"
)

// allowedHosts 记录允许保存页面的原始域名，由启动参数中的种子 URL 决定。
var allowedHosts = make(map[string]bool)

func main() {
	for _, arg := range os.Args[1:] {
		if u, err := url.Parse(arg); err == nil {
			allowedHosts[u.Host] = true
		}
	}
	breadthFirst(crawl, os.Args[1:])
}

func breadthFirst(f func(item string) []string, worklist []string) {
	seen := make(map[string]bool)
	for len(worklist) > 0 {
		items := worklist
		worklist = nil
		for _, item := range items {
			if !seen[item] {
				seen[item] = true
				worklist = append(worklist, f(item)...)
			}
		}
	}
}

func crawl(rawurl string) []string {
	fmt.Println(rawurl)

	if allowedHosts[hostOf(rawurl)] {
		if err := savePage(rawurl); err != nil {
			log.Print(err)
		}
	}

	list, err := links.Extract(rawurl)
	if err != nil {
		log.Print(err)
	}
	return list
}

// hostOf 返回 rawurl 的域名，解析失败时返回空字符串。
func hostOf(rawurl string) string {
	u, err := url.Parse(rawurl)
	if err != nil {
		return ""
	}
	return u.Host
}

// savePage 下载 rawurl 对应的页面并保存到本地磁盘，必要时创建目录。
func savePage(rawurl string) error {
	resp, err := http.Get(rawurl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("getting %s: %s", rawurl, resp.Status)
	}

	path, err := localPath(rawurl)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// localPath 把 URL 映射成本地文件路径，例如：
// https://golang.org/pkg/fmt/ -> golang.org/pkg/fmt/index.html
func localPath(rawurl string) (string, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}

	name := u.Path
	if name == "" || name[len(name)-1] == '/' {
		name += "index.html"
	}

	return filepath.Join(u.Host, filepath.FromSlash(name)), nil
}
