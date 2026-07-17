package main

import (
	"bufio"
	"fmt"
	"os"
)

// 练习 4.9： 编写一个程序wordfreq程序，报告输入文本中每个单词出现的频率。在第一次调用Scan前先调用input.Split(bufio.ScanWords)函数，这样可以按单词而不是按行输入。

func main() {
	freq := make(map[string]int)
	in := bufio.NewScanner(os.Stdin)
	in.Split(bufio.ScanWords)

	for in.Scan() {
		word := in.Text()
		if word == "quit" {
			break
		}
		freq[word]++
	}

	if err := in.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "wordfreq: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("word\tfreq\n")
	for w, c := range freq {
		fmt.Printf("%s\t%d\n", w, c)
	}
}
