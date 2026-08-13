package main

import "io"

// **练习 7.5：** io包里面的LimitReader函数接收一个io.Reader接口类型的r和字节数n，
// 并且返回另一个从r中读取字节但是当读完n个字节后就表示读到文件结束的Reader。实现这个LimitReader函数：

type Reader struct {
	r io.Reader
	n int64 // 剩余可读字节数
}

func (r *Reader) Read(p []byte) (n int, err error) {
	if r.n <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > r.n {
		p = p[:r.n]
	}
	n, err = r.r.Read(p)
	r.n -= int64(n)
	return n, err
}

func LimitReader(r io.Reader, n int64) io.Reader {
	return &Reader{r: r, n: n}
}
