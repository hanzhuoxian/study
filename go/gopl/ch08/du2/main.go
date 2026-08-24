package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var verbose = flag.Bool("v", false, "show verbose progress messages")

// dirents 返回 dir 目录下的所有条目（文件 + 子目录），只读一层。
func dirents(dir string) []os.DirEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "du1: %v\n", err)
		return nil
	}
	return entries
}

// walkDir 递归遍历以 dir 为根的文件树，
// 并在 filesize 上发送每个已找到的文件的大小。
func walkDir(dir string, filesize chan<- int64) {
	for _, entry := range dirents(dir) {
		if entry.IsDir() {
			walkDir(filepath.Join(dir, entry.Name()), filesize)
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
	fmt.Printf("%d files  %.1f GB\n", nfiles, float64(nbytes)/1e9)
}

func main() {
	// 确定初始目录
	flag.Parse()
	roots := flag.Args()
	if len(roots) == 0 {
		roots = []string{"."}
	}

	// 遍历文件树
	filesize := make(chan int64)
	go func() {
		for _, root := range roots {
			walkDir(root, filesize)
		}
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
