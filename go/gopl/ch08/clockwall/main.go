package main

// 练习 8.1（后半）：clockwall 同时与多个 clock 服务器通信，把各地时间显示在一张表格里。
//
//	$ TZ=US/Eastern    ./clock -port 8010 &
//	$ TZ=Asia/Tokyo    ./clock -port 8020 &
//	$ TZ=Europe/London ./clock -port 8030 &
//	$ clockwall NewYork=localhost:8010 Tokyo=localhost:8020 London=localhost:8030

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	offline  = "--:--:--" // 连不上或已断开的时钟显示成这样
	colGap   = "  "       // 列间距
	minWidth = len(offline)
)

// clock 是一台待监视的 clock 服务器。
type clock struct {
	city string
	addr string
}

func main() {
	clocks, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "clockwall: %v\n用法: clockwall 城市=主机:端口 ...\n", err)
		os.Exit(1)
	}
	run(os.Stdout, clocks, time.Second)
}

// update 是某个城市报来的一次新时间。
type update struct {
	city string
	time string
}

// run 为每个时钟起一个 goroutine 去读时间，自己每隔 interval 打印一行表格。
// latest 只被 run 这一个 goroutine 读写，所以不需要锁；goroutine 之间只通过 channel 通信。
// 所有时钟都断开后（channel 关闭）函数返回。
func run(w io.Writer, clocks []clock, interval time.Duration) {
	updates := make(chan update)
	var wg sync.WaitGroup
	for _, c := range clocks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			watch(c, updates)
		}()
	}
	go func() {
		wg.Wait()
		close(updates) // 最后一个时钟也走了，通知主循环收工
	}()

	latest := make(map[string]string, len(clocks))
	for _, c := range clocks {
		latest[c.city] = offline
	}

	widths := make([]int, len(clocks))
	cells := make([]string, len(clocks))
	for i, c := range clocks {
		widths[i] = max(len(c.city), minWidth)
		cells[i] = c.city
	}
	fmt.Fprintln(w, formatRow(cells, widths))

	printRow := func() {
		for i, c := range clocks {
			cells[i] = latest[c.city]
		}
		fmt.Fprintln(w, formatRow(cells, widths))
	}

	tick := time.NewTicker(interval)
	defer tick.Stop()
	for updates != nil {
		select {
		case u, ok := <-updates:
			if !ok {
				updates = nil // 全部时钟都断开了，退出主循环
				continue
			}
			latest[u.city] = u.time
		case <-tick.C:
			printRow()
		}
	}
	// 收尾再打一行。一台都连不上时这是唯一的一行，否则用户只能看到一个光秃秃的表头。
	printRow()
}

// watch 连上一台时钟服务器，把读到的每一行时间送进 updates。
// 连不上或连接断开时报一次 offline 再返回——调用方据此知道这一列已经没人更新了。
func watch(c clock, updates chan<- update) {
	conn, err := net.Dial("tcp", c.addr)
	if err != nil {
		updates <- update{c.city, offline}
		return
	}
	defer conn.Close()

	input := bufio.NewScanner(conn)
	for input.Scan() {
		updates <- update{c.city, strings.TrimSpace(input.Text())}
	}
	updates <- update{c.city, offline}
}

// formatRow 按列宽拼一行，末尾的空白去掉。
func formatRow(cells []string, widths []int) string {
	var b strings.Builder
	for i, cell := range cells {
		if i > 0 {
			b.WriteString(colGap)
		}
		fmt.Fprintf(&b, "%-*s", widths[i], cell)
	}
	return strings.TrimRight(b.String(), " ")
}

// parseArgs 把 "城市=主机:端口" 形式的参数解析成按城市名排序的列表。
// 排序是为了让表格的列固定：map 的遍历顺序是随机的，不排序每行的列都会跳。
func parseArgs(args []string) ([]clock, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("至少需要一个时钟服务器")
	}
	clocks := make([]clock, 0, len(args))
	seen := make(map[string]bool, len(args))
	for _, arg := range args {
		city, addr, ok := strings.Cut(arg, "=")
		if !ok {
			return nil, fmt.Errorf("参数 %q 缺少 '='", arg)
		}
		if city == "" || addr == "" {
			return nil, fmt.Errorf("参数 %q 的城市名和地址都不能为空", arg)
		}
		if seen[city] {
			return nil, fmt.Errorf("城市 %q 重复", city)
		}
		seen[city] = true
		clocks = append(clocks, clock{city: city, addr: addr})
	}
	sort.Slice(clocks, func(i, j int) bool { return clocks[i].city < clocks[j].city })
	return clocks, nil
}
