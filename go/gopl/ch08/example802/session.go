package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// dataTimeout 是等客户端来连数据端口的上限，免得客户端跑了以后 goroutine 一直挂在 Accept 上。
const dataTimeout = 30 * time.Second

// session 是一个控制连接的全部状态。每个连接一个 session，字段只被自己的 goroutine 读写。
type session struct {
	conn net.Conn
	r    *bufio.Reader
	root string           // 服务根目录的绝对路径
	cwd  string           // 当前目录，相对 root 的斜杠路径，始终以 "/" 开头
	data *net.TCPListener // PASV 开出的数据端口监听器，用一次就关
}

func newSession(conn net.Conn, root string) *session {
	return &session{
		conn: conn,
		r:    bufio.NewReader(conn),
		root: root,
		cwd:  "/",
	}
}

// serve 读取并分发命令，直到客户端断开或发送 QUIT。
func (s *session) serve() {
	defer s.conn.Close()
	defer s.closeData()

	s.reply(220, "Service ready")
	for {
		line, err := s.r.ReadString('\n')
		if err != nil {
			return
		}
		cmd, arg := split(strings.TrimRight(line, "\r\n"))
		if cmd == "QUIT" {
			s.reply(221, "Goodbye")
			return
		}
		s.dispatch(cmd, arg)
	}
}

func (s *session) dispatch(cmd, arg string) {
	switch cmd {
	case "USER":
		s.reply(331, "User name okay, need password")
	case "PASS":
		s.reply(230, "Login successful")
	case "PWD", "XPWD":
		s.reply(257, fmt.Sprintf("%q is the current directory", s.cwd))
	case "CWD", "XCWD":
		s.changeDir(arg)
	case "CDUP", "XCUP":
		s.changeDir("..")
	case "PASV":
		s.passive()
	case "LIST":
		s.list(arg, true)
	case "NLST":
		s.list(arg, false)
	case "RETR":
		s.retrieve(arg)
	case "STOR":
		s.store(arg)
	case "PORT", "EPRT", "EPSV":
		// 只实现 PASV 一条数据通道路径，客户端收到 502 会退回 PASV。
		s.reply(502, cmd+" not implemented, use PASV")
	case "TYPE":
		// 字节原样传输，不做 TYPE A 的 CRLF 转换，所以两种模式行为相同。
		switch strings.ToUpper(arg) {
		case "A", "I", "L 8":
			s.reply(200, "Type set to "+arg)
		default:
			s.reply(504, "Unsupported type "+arg)
		}
	case "SYST":
		s.reply(215, "UNIX Type: L8")
	case "FEAT":
		s.reply(211, "No extended features")
	case "NOOP":
		s.reply(200, "NOOP ok")
	default:
		s.reply(500, "Unknown command "+cmd)
	}
}

// errEscape 表示路径越过了根目录。根目录之外的东西对客户端不可见。
var errEscape = errors.New("path escapes root")

// resolve 把客户端给的路径（相对或绝对）解析成虚拟路径和对应的本地路径。
// 它自己走一遍路径分段而不用 path.Clean，因为 Clean 会把 "/../.." 悄悄夹成 "/"，
// 那样逃逸尝试就被吞掉了，这里要的是明确报错。
func (s *session) resolve(arg string) (virt, local string, err error) {
	base := s.cwd
	if strings.HasPrefix(arg, "/") {
		base = "/"
	}
	var stack []string
	for _, elem := range strings.Split(base+"/"+arg, "/") {
		switch elem {
		case "", ".": // 空段和 "." 不改变位置
		case "..":
			if len(stack) == 0 {
				return "", "", errEscape
			}
			stack = stack[:len(stack)-1]
		default:
			stack = append(stack, elem)
		}
	}
	return "/" + strings.Join(stack, "/"), filepath.Join(s.root, filepath.Join(stack...)), nil
}

// changeDir 处理 CWD/CDUP。
func (s *session) changeDir(arg string) {
	if arg == "" {
		s.reply(501, "Syntax error: CWD needs a path")
		return
	}
	virt, local, err := s.resolve(arg)
	if err != nil {
		s.reply(550, arg+": Permission denied")
		return
	}
	fi, err := os.Stat(local)
	if err != nil {
		s.reply(550, arg+": No such file or directory")
		return
	}
	if !fi.IsDir() {
		s.reply(550, arg+": Not a directory")
		return
	}
	s.cwd = virt
	s.reply(250, "Directory changed to "+virt)
}

// passive 处理 PASV：开一个临时端口等客户端来连。
// 端口报在应答里，取的是控制连接的本地地址，这样多网卡机器上客户端也能连回来。
func (s *session) passive() {
	s.closeData() // 上一次 PASV 没用掉的监听器先关掉

	host := s.conn.LocalAddr().(*net.TCPAddr).IP
	ip := host.To4()
	if ip == nil {
		s.reply(522, "Network protocol not supported, use IPv4")
		return
	}
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: host})
	if err != nil {
		s.reply(425, "Can't open data connection: "+err.Error())
		return
	}
	s.data = l

	port := l.Addr().(*net.TCPAddr).Port
	s.reply(227, fmt.Sprintf("Entering Passive Mode (%d,%d,%d,%d,%d,%d)",
		ip[0], ip[1], ip[2], ip[3], port>>8, port&0xff))
}

func (s *session) closeData() {
	if s.data != nil {
		s.data.Close()
		s.data = nil
	}
}

// transfer 是所有数据命令的公共流程：确认有数据连接 -> 150 -> 传输 -> 226。
// 数据连接每条命令用一次，用完就关，所以客户端每次传输前都得重新 PASV。
func (s *session) transfer(msg string, do func(net.Conn) error) {
	if s.data == nil {
		s.reply(425, "Use PASV first")
		return
	}
	defer s.closeData()

	s.reply(150, msg)
	s.data.SetDeadline(time.Now().Add(dataTimeout))
	conn, err := s.data.Accept()
	if err != nil {
		s.reply(425, "Can't open data connection: "+err.Error())
		return
	}
	err = do(conn)
	conn.Close() // 关闭数据连接就是"传输结束"的信号，必须发生在 226 之前
	if err != nil {
		s.reply(426, "Transfer aborted: "+err.Error())
		return
	}
	s.reply(226, "Transfer complete")
}

// list 处理 LIST（长格式）和 NLST（只有文件名）。
func (s *session) list(arg string, long bool) {
	_, local, err := s.resolve(arg)
	if err != nil {
		s.reply(550, "Permission denied")
		return
	}
	entries, err := readEntries(local)
	if err != nil {
		s.reply(550, "Can't list "+arg+": "+err.Error())
		return
	}
	s.transfer("Here comes the directory listing", func(conn net.Conn) error {
		w := bufio.NewWriter(conn)
		for _, e := range entries {
			line := e.Name()
			if long {
				fi, err := e.Info()
				if err != nil {
					continue // 列表期间文件被删了，跳过
				}
				line = formatEntry(fi)
			}
			if _, err := fmt.Fprintf(w, "%s\r\n", line); err != nil {
				return err
			}
		}
		return w.Flush()
	})
}

// readEntries 列出一个路径：目录列其内容，普通文件只列自己（客户端的 ls FILE 会这么用）。
func readEntries(local string) ([]os.DirEntry, error) {
	fi, err := os.Stat(local)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return os.ReadDir(local)
	}
	return []os.DirEntry{fs.FileInfoToDirEntry(fi)}, nil
}

// retrieve 处理 RETR：把文件内容写到数据连接上。
func (s *session) retrieve(arg string) {
	local, ok := s.dataPath(arg)
	if !ok {
		return
	}
	f, err := os.Open(local)
	if err != nil {
		s.reply(550, arg+": "+errText(err))
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || fi.IsDir() {
		s.reply(550, arg+": Not a regular file")
		return
	}
	s.transfer(fmt.Sprintf("Opening data connection for %s (%d bytes)", arg, fi.Size()),
		func(conn net.Conn) error {
			_, err := io.Copy(conn, f)
			return err
		})
}

// store 处理 STOR：把数据连接上收到的内容写进文件。
func (s *session) store(arg string) {
	local, ok := s.dataPath(arg)
	if !ok {
		return
	}
	f, err := os.Create(local)
	if err != nil {
		s.reply(550, arg+": "+errText(err))
		return
	}
	s.transfer("Ok to send data", func(conn net.Conn) error {
		_, err := io.Copy(f, conn)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		return err
	})
}

// dataPath 解析 RETR/STOR 的路径参数，参数或路径有问题时自己回好应答并返回 false。
// 这些检查要在 150 之前做完，客户端才不会先看到"开始传输"再看到失败。
func (s *session) dataPath(arg string) (local string, ok bool) {
	if arg == "" {
		s.reply(501, "Syntax error: a path is required")
		return "", false
	}
	_, local, err := s.resolve(arg)
	if err != nil {
		s.reply(550, arg+": Permission denied")
		return "", false
	}
	return local, true
}

// errText 把 *os.PathError 里的系统错误抽出来，避免把服务器的本地路径回给客户端。
func errText(err error) string {
	var perr *os.PathError
	if errors.As(err, &perr) {
		return perr.Err.Error()
	}
	return err.Error()
}

// formatEntry 生成类 Unix 的长格式列表行，客户端的 ls 就靠解析它来显示。
func formatEntry(fi os.FileInfo) string {
	return fmt.Sprintf("%s 1 ftp ftp %8d %s %s",
		fi.Mode(), fi.Size(), fi.ModTime().Format("Jan _2 15:04"), fi.Name())
}

// reply 写一条单行应答。协议规定行以 CRLF 结尾。
func (s *session) reply(code int, msg string) {
	if _, err := fmt.Fprintf(s.conn, "%d %s\r\n", code, msg); err != nil {
		log.Printf("写应答失败: %v", err)
	}
}

// split 把一行命令拆成大写的动词和参数。
func split(line string) (cmd, arg string) {
	line = strings.TrimSpace(line)
	if i := strings.IndexByte(line, ' '); i >= 0 {
		return strings.ToUpper(line[:i]), strings.TrimSpace(line[i+1:])
	}
	return strings.ToUpper(line), ""
}
