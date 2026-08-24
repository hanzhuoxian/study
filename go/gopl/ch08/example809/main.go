package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// **练习 8.9：** 编写一个du工具，每隔一段时间将root目录下的目录大小计算并显示出来。

var verbose = flag.Bool("v", false, "show verbose progress messages")
var t = flag.Duration("t", time.Minute, "time")

var count atomic.Int64

var sema = make(chan struct{}, 200)

func main() {
	start := time.Now()

	// 确定初始目录
	flag.Parse()
	roots := flag.Args()
	if len(roots) == 0 {
		roots = []string{"."}
	}
	tick := time.Tick(*t)
	for {
		fmt.Println(time.Now())
		run(roots)
		fmt.Printf("elapsed time: %v\n", time.Since(start))
		<-tick
	}
}

// dirents 返回 dir 目录下的所有条目（文件 + 子目录），只读一层。
func dirents(dir string) []os.DirEntry {
	sema <- struct{}{}
	count.Add(1)
	defer func() {
		<-sema
		count.Add(-1)
	}()
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "du1: %v\n", err)
		return nil
	}
	return entries
}

// walkDir 递归遍历以 dir 为根的文件树，
// 并在 filesize 上发送每个已找到的文件的大小。
func walkDir(dir string, n *sync.WaitGroup, filesize chan<- int64) {
	defer n.Done()
	for _, entry := range dirents(dir) {
		if entry.IsDir() {
			n.Add(1)
			go walkDir(filepath.Join(dir, entry.Name()), n, filesize)
		} else {
			info, err := entry.Info()
			if err != nil {
				fmt.Fprintf(os.Stderr, "du1: %v\n", err)
				continue
			}
			filesize <- info.Size()
		}
	}
}

func printDiskUsage(nfiles, nbytes int64) {
	fmt.Printf("%d files  %.1f GB count %d\n", nfiles, float64(nbytes)/1e9, count.Load())
}

func run(roots []string) {
	// 遍历文件树
	filesize := make(chan int64)
	var n sync.WaitGroup
	// n.Add 必须在 n.Wait 之前完成，所以这里在 main goroutine 里同步累加计数
	for _, root := range roots {
		n.Add(1)
		go walkDir(root, &n, filesize)
	}

	go func() {
		n.Wait()
		close(filesize)
	}()

	var tick <-chan time.Time
	if *verbose {
		tick = time.Tick(500 * time.Millisecond)
	}
	var nfiles, nbytes int64
loop:
	for {
		select {
		case <-tick:
			printDiskUsage(nfiles, nbytes)
		case size, ok := <-filesize:
			if !ok {
				break loop
			}
			nfiles++
			nbytes += size
		}
	}
	printDiskUsage(nfiles, nbytes)
}
