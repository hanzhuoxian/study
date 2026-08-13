package main

import (
	"bufio"
	"bytes"
	"fmt"
)

// **练习 7.1：** 使用来自ByteCounter的思路，实现一个针对单词和行数的计数器。你会发现bufio.ScanWords非常的有用。

type Counter struct {
	buf  bytes.Buffer
	word int
	line int
}

// Write 把新数据追加到内部缓冲区，再对累积的全部内容重新扫描计数。
// 这样即使一行/一个单词被拆到前后两次 Write 调用里，也能正确统计，
// 不会因为按单次调用独立扫描而漏算或重复计算。
func (c *Counter) Write(b []byte) (n int, err error) {
	n, err = c.buf.Write(b)
	if err != nil {
		return n, err
	}

	c.line = 0
	lineScan := bufio.NewScanner(bytes.NewReader(c.buf.Bytes()))
	lineScan.Split(bufio.ScanLines)
	for lineScan.Scan() {
		c.line++
	}

	c.word = 0
	wordScan := bufio.NewScanner(bytes.NewReader(c.buf.Bytes()))
	wordScan.Split(bufio.ScanWords)
	for wordScan.Scan() {
		c.word++
	}
	return n, nil
}

// GoString 让 %#v 只打印 word/line，避免把内部 bytes.Buffer 的细节也输出出来。
func (c Counter) GoString() string {
	return fmt.Sprintf("Counter{word:%d, line:%d}", c.word, c.line)
}

func main() {
	var c Counter
	fmt.Fprint(&c, "Hello world!\n Hello world!")
	fmt.Fprint(&c, `Hello world!
		Hello world!`)
	fmt.Printf("%#v", c)
}
