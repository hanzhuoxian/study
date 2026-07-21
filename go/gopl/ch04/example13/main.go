package main

import (
	"bufio"
	"fmt"
	"os"
)

// 练习 4.13： 使用开放电影数据库的JSON服务接口，允许你检索和下载 https://omdbapi.com/ 上电影的名字和对应的海报图像。编写一个poster工具，通过命令行输入的电影名字，下载对应的海报。

func main() {
	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		line := input.Text()
		if line == "quit" {
			fmt.Println("quit")
			break
		}

		movies, err := search(line)
		if err != nil {

		}

		for _, movie := range movies {
			err := download(movie.Url)
			if err != nil {

			}
		}
	}
}

type movie struct {
	Url string
}

func search(name string) ([]*movie, error) {
	return nil, nil
}

func download(url string) error {
	return nil
}
