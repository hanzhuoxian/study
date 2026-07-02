package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func main() {
	// 定义缓存行次数的map
	counts := make(map[string]map[string]int)
	// 获取命令行
	files := os.Args[1:]
	if len(files) == 0 {
		// 使用标准输入
		fmt.Println("Please input your text: ")
		countLines("stdin", os.Stdin, counts)
	} else {
		// 循环读取文件
		for _, filePath := range files {
			// 打开文件
			file, err := os.Open(filePath)
			if err != nil {
				log.Printf("open %s is failed", filePath)
				continue
			}
			countLines(filePath, file, counts)
			file.Close()
		}
	}

	for filename, fcounts := range counts {

		for line, n := range fcounts {
			if n > 1 {
				fmt.Printf("%s\t%d\t%s\n", filename, n, line)
			}
		}
	}

}

func countLines(filename string, f *os.File, counts map[string]map[string]int) {
	input := bufio.NewScanner(f)
	counts[filename] = make(map[string]int)
	for input.Scan() {
		counts[filename][input.Text()]++
	}
	if input.Err() != nil {
		log.Printf("open %v is failed", input.Err())
	}
}
