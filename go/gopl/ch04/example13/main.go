package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
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
			log.Fatal(err)
		}

		for _, movie := range movies {
			err := download(movie.Url, movie.Name)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	if err := input.Err(); err != nil {
		log.Fatal(err)
	}
}

const PosterURL = ""

type Movie struct {
	Url  string
	Name string
}

func search(name string) ([]*Movie, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(PosterURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var m []*Movie
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &m)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func download(url, filepath string) error {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Http 错误 %d %s", resp.StatusCode, resp.Status)
	}

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("create file failed : %w", err)
	}
	defer file.Close()

	n, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("write file failed %w", err)
	}

	fmt.Printf("download success %s (%d bytes)\n", filepath, n)

	return err
}
