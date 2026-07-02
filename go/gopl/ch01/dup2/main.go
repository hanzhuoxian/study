package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

//下面的dup程序从 标准输入得到一些文件名，然后用os.Open函数来打开每一个文件获取内容。

func main() {
	// 定义缓存行次数的map
	counts := make(map[string]int)
	// 获取命令行
	files := os.Args[1:]
	if len(files) == 0 {
		// 使用标准输入
		fmt.Println("Please input your text: ")
		countLines(os.Stdin, counts)
	} else {
		// 循环读取文件
		for _, filePath := range files {
			// 打开文件
			file, err := os.Open(filePath)
			if err != nil {
				log.Printf("open %s is failed", filePath)
				continue
			}
			countLines(file, counts)
			file.Close()
		}
	}

	for line, n := range counts {
		if n > 1 {
			fmt.Printf("%d\t%s\n", n, line)
		}
	}

}

func countLines(f *os.File, counts map[string]int) {
	input := bufio.NewScanner(f)
	for input.Scan() {
		counts[input.Text()]++
	}
	if input.Err() != nil {
		log.Printf("open %v is failed", input.Err())
	}
}
