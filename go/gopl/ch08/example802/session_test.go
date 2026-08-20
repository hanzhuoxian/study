package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// ---- 测试脚手架：真实 socket 上的服务器 + 手写的最小 FTP 客户端 ----

// startServer 在临时目录上起一台服务器，返回监听地址和根目录。
func startServer(t *testing.T) (addr, root string) {
	t.Helper()
	root = t.TempDir()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go serve(l, root)
	return l.Addr().String(), root
}

type client struct {
	t    *testing.T
	conn net.Conn
	r    *bufio.Reader
}

// dial 建立控制连接并消费欢迎语。
func dial(t *testing.T, addr string) *client {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	c := &client{t: t, conn: conn, r: bufio.NewReader(conn)}
	c.expect(220)
	return c
}

// reply 读一条应答，支持 "xxx-" 开头的多行应答。
func (c *client) reply() (int, string) {
	c.t.Helper()
	line := c.line()
	if len(line) < 4 {
		c.t.Fatalf("应答格式非法: %q", line)
	}
	code, err := strconv.Atoi(line[:3])
	if err != nil {
		c.t.Fatalf("应答格式非法: %q", line)
	}
	if line[3] == '-' { // 多行应答：读到 "xxx " 收尾行为止
		for {
			next := c.line()
			if strings.HasPrefix(next, line[:3]+" ") {
				return code, next[4:]
			}
		}
	}
	return code, line[4:]
}

func (c *client) line() string {
	c.t.Helper()
	line, err := c.r.ReadString('\n')
	if err != nil {
		c.t.Fatalf("读取应答失败: %v", err)
	}
	if !strings.HasSuffix(line, "\r\n") {
		c.t.Errorf("应答未以 CRLF 结尾: %q", line)
	}
	return strings.TrimRight(line, "\r\n")
}

func (c *client) cmd(format string, args ...any) (int, string) {
	c.t.Helper()
	fmt.Fprintf(c.conn, format+"\r\n", args...)
	return c.reply()
}

// expect 断言下一条应答的状态码。
func (c *client) expect(want int) string {
	c.t.Helper()
	code, msg := c.reply()
	if code != want {
		c.t.Fatalf("期望应答码 %d，实际 %d %s", want, code, msg)
	}
	return msg
}

// mustCmd 发一条命令并断言状态码。
func (c *client) mustCmd(want int, format string, args ...any) string {
	c.t.Helper()
	code, msg := c.cmd(format, args...)
	if code != want {
		c.t.Fatalf("%s: 期望应答码 %d，实际 %d %s", fmt.Sprintf(format, args...), want, code, msg)
	}
	return msg
}

// pasv 进入被动模式，按 227 应答里的地址建立数据连接。
func (c *client) pasv() net.Conn {
	c.t.Helper()
	msg := c.mustCmd(227, "PASV")
	lp, rp := strings.IndexByte(msg, '('), strings.LastIndexByte(msg, ')')
	if lp < 0 || rp < lp {
		c.t.Fatalf("227 应答里找不到地址: %s", msg)
	}
	parts := strings.Split(msg[lp+1:rp], ",")
	if len(parts) != 6 {
		c.t.Fatalf("227 地址应为 6 个数字: %s", msg)
	}
	var n [6]int
	for i, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			c.t.Fatalf("227 地址非法: %s", msg)
		}
		n[i] = v
	}
	addr := net.JoinHostPort(
		fmt.Sprintf("%d.%d.%d.%d", n[0], n[1], n[2], n[3]),
		strconv.Itoa(n[4]<<8|n[5]))
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		c.t.Fatalf("连接数据端口 %s 失败: %v", addr, err)
	}
	c.t.Cleanup(func() { conn.Close() })
	return conn
}

// download 跑一条下行数据命令（LIST/NLST/RETR），返回数据连接上收到的全部字节。
func (c *client) download(format string, args ...any) string {
	c.t.Helper()
	data := c.pasv()
	defer data.Close()
	c.mustCmd(150, format, args...)
	b, err := io.ReadAll(data)
	if err != nil {
		c.t.Fatalf("读数据连接失败: %v", err)
	}
	c.expect(226)
	return string(b)
}

// upload 跑一条上行数据命令（STOR），把内容写进数据连接。
func (c *client) upload(content, format string, args ...any) {
	c.t.Helper()
	data := c.pasv()
	c.mustCmd(150, format, args...)
	if _, err := data.Write([]byte(content)); err != nil {
		c.t.Fatalf("写数据连接失败: %v", err)
	}
	data.Close() // 关闭数据连接就是上传结束的信号
	c.expect(226)
}

func (c *client) login() {
	c.t.Helper()
	c.mustCmd(331, "USER anonymous")
	c.mustCmd(230, "PASS whatever")
}

// ---- 测试 ----

func TestNewConnectionGetsGreeting(t *testing.T) {
	addr, _ := startServer(t)
	dial(t, addr) // dial 内部断言 220
}

func TestPWDReportsRootAfterLogin(t *testing.T) {
	addr, _ := startServer(t)
	c := dial(t, addr)
	c.login()
	msg := c.mustCmd(257, "PWD")
	if !strings.Contains(msg, `"/"`) {
		t.Errorf("PWD 应报告根目录 \"/\"，实际: %s", msg)
	}
}

func TestUnknownCommandGets500(t *testing.T) {
	addr, _ := startServer(t)
	c := dial(t, addr)
	c.login()
	c.mustCmd(500, "FLY TO THE MOON")
}

func TestQuitClosesConnection(t *testing.T) {
	addr, _ := startServer(t)
	c := dial(t, addr)
	c.login()
	c.mustCmd(221, "QUIT")
	if _, err := c.r.ReadString('\n'); err == nil {
		t.Error("QUIT 之后连接应被关闭")
	}
}

func TestCWDChangesDirectory(t *testing.T) {
	addr, root := startServer(t)
	if err := os.MkdirAll(filepath.Join(root, "docs", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := dial(t, addr)
	c.login()
	c.mustCmd(250, "CWD docs")
	if msg := c.mustCmd(257, "PWD"); !strings.Contains(msg, `"/docs"`) {
		t.Errorf(`期望 cwd 为 "/docs"，实际: %s`, msg)
	}
	c.mustCmd(250, "CWD deep") // 相对当前目录
	if msg := c.mustCmd(257, "PWD"); !strings.Contains(msg, `"/docs/deep"`) {
		t.Errorf(`期望 cwd 为 "/docs/deep"，实际: %s`, msg)
	}
	c.mustCmd(250, "CWD /docs") // 绝对路径
	if msg := c.mustCmd(257, "PWD"); !strings.Contains(msg, `"/docs"`) {
		t.Errorf(`期望 cwd 为 "/docs"，实际: %s`, msg)
	}
}

func TestCDUPGoesToParent(t *testing.T) {
	addr, root := startServer(t)
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := dial(t, addr)
	c.login()
	c.mustCmd(250, "CWD docs")
	c.mustCmd(250, "CDUP")
	if msg := c.mustCmd(257, "PWD"); !strings.Contains(msg, `"/"`) {
		t.Errorf(`期望 cwd 回到 "/"，实际: %s`, msg)
	}
	c.mustCmd(550, "CDUP") // 根目录已无父目录可去，与 CWD .. 一致地拒绝
}

func TestCWDToMissingDirectoryFails(t *testing.T) {
	addr, _ := startServer(t)
	c := dial(t, addr)
	c.login()
	c.mustCmd(550, "CWD nope")
}

func TestCWDToRegularFileFails(t *testing.T) {
	addr, root := startServer(t)
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := dial(t, addr)
	c.login()
	c.mustCmd(550, "CWD a.txt")
}

func TestPathEscapeIsRejected(t *testing.T) {
	addr, root := startServer(t)
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := dial(t, addr)
	c.login()
	for _, path := range []string{"..", "../..", "/../..", "docs/../.."} {
		if code, msg := c.cmd("CWD %s", path); code != 550 {
			t.Errorf("CWD %s 应被拒绝，实际: %d %s", path, code, msg)
		}
	}
}

func TestCWDWithoutArgumentGets501(t *testing.T) {
	addr, _ := startServer(t)
	c := dial(t, addr)
	c.login()
	c.mustCmd(501, "CWD")
}

func TestListShowsDirectoryEntries(t *testing.T) {
	addr, root := startServer(t)
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := dial(t, addr)
	c.login()
	listing := c.download("LIST")

	var sawFile, sawDir bool
	for _, line := range strings.Split(strings.TrimSpace(listing), "\r\n") {
		switch {
		case strings.HasSuffix(line, " a.txt"):
			sawFile = true
			if !strings.HasPrefix(line, "-") {
				t.Errorf("普通文件行应以 - 开头: %q", line)
			}
			if !strings.Contains(line, "5") {
				t.Errorf("文件行应含大小 5: %q", line)
			}
		case strings.HasSuffix(line, " docs"):
			sawDir = true
			if !strings.HasPrefix(line, "d") {
				t.Errorf("目录行应以 d 开头: %q", line)
			}
		}
	}
	if !sawFile || !sawDir {
		t.Errorf("LIST 应同时列出 a.txt 和 docs，实际:\n%s", listing)
	}
}

func TestListRespectsCurrentDirectory(t *testing.T) {
	addr, root := startServer(t)
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "inner.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := dial(t, addr)
	c.login()
	c.mustCmd(250, "CWD docs")
	if listing := c.download("LIST"); !strings.Contains(listing, "inner.txt") {
		t.Errorf("LIST 应列出当前目录内容，实际:\n%s", listing)
	}
}

func TestNLSTShowsBareNames(t *testing.T) {
	addr, root := startServer(t)
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := dial(t, addr)
	c.login()
	if got := c.download("NLST"); got != "a.txt\r\n" {
		t.Errorf("NLST 应只输出文件名，实际: %q", got)
	}
}

func TestDataCommandWithoutPasvGets425(t *testing.T) {
	addr, _ := startServer(t)
	c := dial(t, addr)
	c.login()
	c.mustCmd(425, "LIST")
}

func TestActiveModeAndEPSVAreRejected(t *testing.T) {
	addr, _ := startServer(t)
	c := dial(t, addr)
	c.login()
	c.mustCmd(502, "PORT 127,0,0,1,4,0")
	c.mustCmd(502, "EPSV")
}

func TestRetrReturnsFileContentByteForByte(t *testing.T) {
	addr, root := startServer(t)
	content := "line one\nline two\r\n\x00\xff binary"
	if err := os.WriteFile(filepath.Join(root, "a.bin"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	c := dial(t, addr)
	c.login()
	if got := c.download("RETR a.bin"); got != content {
		t.Errorf("RETR 内容应逐字节一致\n期望: %q\n实际: %q", content, got)
	}
}

func TestRetrResolvesRelativeToCurrentDirectory(t *testing.T) {
	addr, root := startServer(t)
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "inner.txt"), []byte("inner"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := dial(t, addr)
	c.login()
	c.mustCmd(250, "CWD docs")
	if got := c.download("RETR inner.txt"); got != "inner" {
		t.Errorf("期望 inner，实际 %q", got)
	}
}

func TestRetrMissingFileGets550(t *testing.T) {
	addr, _ := startServer(t)
	c := dial(t, addr)
	c.login()
	c.pasv()
	c.mustCmd(550, "RETR nope.txt")
}

func TestRetrOnDirectoryGets550(t *testing.T) {
	addr, root := startServer(t)
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := dial(t, addr)
	c.login()
	c.pasv()
	c.mustCmd(550, "RETR docs")
}

func TestStorWritesFile(t *testing.T) {
	addr, root := startServer(t)
	c := dial(t, addr)
	c.login()
	c.upload("uploaded content\n", "STOR new.txt")

	got, err := os.ReadFile(filepath.Join(root, "new.txt"))
	if err != nil {
		t.Fatalf("STOR 后文件应存在: %v", err)
	}
	if string(got) != "uploaded content\n" {
		t.Errorf("落盘内容不符: %q", got)
	}
}

func TestStorRejectsPathEscape(t *testing.T) {
	addr, _ := startServer(t)
	c := dial(t, addr)
	c.login()
	c.pasv()
	c.mustCmd(550, "STOR ../evil.txt")
}

func TestRetrWithoutArgumentGets501(t *testing.T) {
	addr, _ := startServer(t)
	c := dial(t, addr)
	c.login()
	c.pasv()
	c.mustCmd(501, "RETR")
}

func TestListOnSingleFileShowsThatFile(t *testing.T) {
	addr, root := startServer(t)
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c := dial(t, addr)
	c.login()
	listing := c.download("LIST a.txt")
	if !strings.Contains(listing, "a.txt") || strings.Contains(listing, "b.txt") {
		t.Errorf("LIST a.txt 应只列出 a.txt，实际:\n%s", listing)
	}
}

// 本题的正题：每个连接的状态互不干扰。
func TestConcurrentSessionsHaveIndependentCwd(t *testing.T) {
	addr, root := startServer(t)
	for _, dir := range []string{"alpha", "beta"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, dir, dir+".txt"), []byte(dir), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	a, b := dial(t, addr), dial(t, addr)
	a.login()
	b.login()
	a.mustCmd(250, "CWD alpha")
	b.mustCmd(250, "CWD beta")

	// 交错着操作两个会话，任何共享的当前目录都会在这里露馅。
	if msg := a.mustCmd(257, "PWD"); !strings.Contains(msg, `"/alpha"`) {
		t.Errorf(`会话 A 的 cwd 应为 "/alpha"，实际: %s`, msg)
	}
	if msg := b.mustCmd(257, "PWD"); !strings.Contains(msg, `"/beta"`) {
		t.Errorf(`会话 B 的 cwd 应为 "/beta"，实际: %s`, msg)
	}
	if got := a.download("RETR alpha.txt"); got != "alpha" {
		t.Errorf("会话 A 期望读到 alpha，实际 %q", got)
	}
	if got := b.download("RETR beta.txt"); got != "beta" {
		t.Errorf("会话 B 期望读到 beta，实际 %q", got)
	}
	if listing := a.download("LIST"); !strings.Contains(listing, "alpha.txt") || strings.Contains(listing, "beta.txt") {
		t.Errorf("会话 A 的 LIST 应只看到 alpha 目录，实际:\n%s", listing)
	}
}

func TestManyConcurrentTransfers(t *testing.T) {
	addr, root := startServer(t)
	const n = 16
	for i := range n {
		dir := fmt.Sprintf("d%d", i)
		if err := os.Mkdir(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, dir, "f.txt"), []byte(dir), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dir := fmt.Sprintf("d%d", i)
			c := dial(t, addr)
			c.login()
			c.mustCmd(250, "CWD %s", dir)
			c.upload(dir+" uploaded", "STOR up.txt")
			if got := c.download("RETR f.txt"); got != dir {
				t.Errorf("%s: 期望 %q，实际 %q", dir, dir, got)
			}
			c.mustCmd(221, "QUIT")
		}()
	}
	wg.Wait()

	for i := range n {
		dir := fmt.Sprintf("d%d", i)
		got, err := os.ReadFile(filepath.Join(root, dir, "up.txt"))
		if err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
		if string(got) != dir+" uploaded" {
			t.Errorf("%s: 上传内容错位: %q", dir, got)
		}
	}
}
