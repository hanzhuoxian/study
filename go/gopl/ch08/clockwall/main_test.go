package main

import (
	"bytes"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestParseArgsSortsByCity(t *testing.T) {
	clocks, err := parseArgs([]string{"Tokyo=localhost:8020", "London=:8030", "NewYork=127.0.0.1:8010"})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range clocks {
		got = append(got, c.city+"@"+c.addr)
	}
	want := []string{"London@:8030", "NewYork@127.0.0.1:8010", "Tokyo@localhost:8020"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("应按城市名排序\n期望: %v\n实际: %v", want, got)
	}
}

func TestParseArgsRejectsBadInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"没有参数", nil},
		{"缺少等号", []string{"Tokyo"}},
		{"城市名为空", []string{"=:8010"}},
		{"地址为空", []string{"Tokyo="}},
		{"城市名重复", []string{"Tokyo=:8010", "Tokyo=:8020"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseArgs(tt.args); err == nil {
				t.Errorf("parseArgs(%q) 应报错", tt.args)
			}
		})
	}
}

// ---- 测试脚手架：假 clock 服务器 ----

// startClock 起一个假 clock 服务器，每 5ms 写一行固定时间戳，写满 lines 行后主动断开。
func startClock(t *testing.T, stamp string, lines int) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return // 监听器已关闭
			}
			go func() {
				defer conn.Close()
				for range lines {
					if _, err := io.WriteString(conn, stamp+"\n"); err != nil {
						return
					}
					time.Sleep(5 * time.Millisecond)
				}
			}()
		}
	}()
	return l.Addr().String()
}

// deadAddr 返回一个确定没人监听的地址。
func deadAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

// runUntilDone 跑 run 并等它返回，返回它写出的全部内容。
func runUntilDone(t *testing.T, clocks []clock, interval time.Duration) string {
	t.Helper()
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		run(&buf, clocks, interval)
		close(done)
	}()
	select {
	case <-done: // close(done) 之后读 buf 是安全的
	case <-time.After(3 * time.Second):
		t.Fatal("所有服务器都断开后 run 应该返回，但 3s 内没有")
	}
	return buf.String()
}

// rows 拆出表头之后的数据行。
func rows(out string) (header string, data []string) {
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) == 0 {
		return "", nil
	}
	return lines[0], lines[1:]
}

// ---- run 的测试 ----

func TestRunPrintsHeaderWithCitiesInColumnOrder(t *testing.T) {
	clocks := []clock{
		{"London", startClock(t, "17:00:00", 3)},
		{"Tokyo", startClock(t, "01:00:00", 3)},
	}
	header, _ := rows(runUntilDone(t, clocks, 10*time.Millisecond))
	london, tokyo := strings.Index(header, "London"), strings.Index(header, "Tokyo")
	if london < 0 || tokyo < 0 {
		t.Fatalf("表头应含两个城市名，实际: %q", header)
	}
	if london > tokyo {
		t.Errorf("表头列顺序应与 clocks 一致（London 在前），实际: %q", header)
	}
}

func TestRunShowsLatestTimeFromEachServer(t *testing.T) {
	clocks := []clock{
		{"London", startClock(t, "17:00:00", 20)},
		{"Tokyo", startClock(t, "01:00:00", 20)},
	}
	_, data := rows(runUntilDone(t, clocks, 10*time.Millisecond))
	var ok bool
	for _, row := range data {
		if strings.Contains(row, "17:00:00") && strings.Contains(row, "01:00:00") {
			ok = true
		}
	}
	if !ok {
		t.Errorf("应有一行同时含两地时间，实际:\n%s", strings.Join(data, "\n"))
	}
}

func TestRunMarksUnreachableServerOffline(t *testing.T) {
	clocks := []clock{
		{"Dead", deadAddr(t)},
		{"London", startClock(t, "17:00:00", 20)},
	}
	_, data := rows(runUntilDone(t, clocks, 10*time.Millisecond))
	// 最后一行是全部离线的收尾状态，所以这里找的是运行期间的行：
	// Dead 那列一直 --:--:--，而另一台照常报时。
	var ok bool
	for _, row := range data {
		if strings.HasPrefix(row, "--:--:--") && strings.Contains(row, "17:00:00") {
			ok = true
		}
	}
	if !ok {
		t.Errorf("连不上的那列应为 --:--:--，另一台不受影响，实际:\n%s", strings.Join(data, "\n"))
	}
}

func TestRunMarksDisconnectedServerOffline(t *testing.T) {
	clocks := []clock{
		{"Brief", startClock(t, "09:00:00", 2)}, // 写两行就断
		{"London", startClock(t, "17:00:00", 40)},
	}
	_, data := rows(runUntilDone(t, clocks, 10*time.Millisecond))
	var sawOfflineWithLondon bool
	for _, row := range data {
		if strings.Contains(row, "--:--:--") && strings.Contains(row, "17:00:00") {
			sawOfflineWithLondon = true
		}
	}
	if !sawOfflineWithLondon {
		t.Errorf("中途断开的服务器该列应转为 --:--:--，而另一列继续走，实际:\n%s", strings.Join(data, "\n"))
	}
}

func TestRunReturnsWhenAllServersDisconnect(t *testing.T) {
	clocks := []clock{{"Brief", startClock(t, "09:00:00", 1)}}
	runUntilDone(t, clocks, 10*time.Millisecond) // 超时会在 helper 里 Fatal
}

func TestRunPrintsFinalRowWhenAllClocksGoOffline(t *testing.T) {
	clocks := []clock{
		{"Dead1", deadAddr(t)},
		{"Dead2", deadAddr(t)},
	}
	// interval 故意设得比 run 的存活时间长：一台都连不上时，靠 ticker 是等不到任何一行的，
	// 但用户至少该看到一行 --:--:-- 才知道发生了什么。
	_, data := rows(runUntilDone(t, clocks, time.Hour))
	if len(data) != 1 {
		t.Fatalf("应正好输出一行收尾状态，实际 %d 行:\n%s", len(data), strings.Join(data, "\n"))
	}
	if strings.Count(data[0], "--:--:--") != 2 {
		t.Errorf("收尾行应把两列都标成 --:--:--，实际: %q", data[0])
	}
}
