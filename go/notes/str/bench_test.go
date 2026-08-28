package main

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
	"unsafe"
)

// go test -bench . -benchmem ./str
// 只看零拷贝特例：go test -bench Zero -benchmem ./str

var (
	strSink   string
	intSink   int
	boolSink  bool
	bytesSink []byte
	runesSink []rune
)

var (
	short  = "hello world"
	long   = strings.Repeat("hello world ", 1000) // 12KB
	shortB = []byte(short)
	longB  = []byte(long)
)

// ---------------------------------------------------------------------------
// ① 转换成本
// ---------------------------------------------------------------------------

func BenchmarkStringToBytesShort(b *testing.B) {
	for b.Loop() {
		bytesSink = []byte(short)
	}
}

func BenchmarkStringToBytesLong(b *testing.B) {
	for b.Loop() {
		bytesSink = []byte(long)
	}
}

func BenchmarkBytesToStringLong(b *testing.B) {
	for b.Loop() {
		strSink = string(longB)
	}
}

func BenchmarkStringToRunesLong(b *testing.B) {
	for b.Loop() {
		runesSink = []rune(long)
	}
}

func BenchmarkBytesToStringUnsafeLong(b *testing.B) {
	for b.Loop() {
		strSink = unsafe.String(unsafe.SliceData(longB), len(longB))
	}
}

// ---------------------------------------------------------------------------
// ② 编译器的零拷贝特例
//
// 实测结论（go1.26.3）：
//   ZeroCopyMapLookup   10.09 ns/op   0 B/op  0 allocs/op   ✓ 真零拷贝
//   ZeroCopyCompare      1.09 ns/op   0 B/op  0 allocs/op   ✓ 真零拷贝
//   ZeroCopyRange    31349    ns/op  12288 B/op  1 allocs/op ✗ 会拷贝！
//   NotZeroCopy          7.77 ns/op   0 B/op  0 allocs/op   小字符串走栈上 tmpBuf
// ---------------------------------------------------------------------------

var m = map[string]int{"hello world": 1}

func BenchmarkZeroCopyMapLookup(b *testing.B) {
	for b.Loop() {
		intSink = m[string(shortB)]
	}
}

func BenchmarkZeroCopyCompare(b *testing.B) {
	for b.Loop() {
		boolSink = string(shortB) == "hello world"
	}
}

func BenchmarkZeroCopyRange(b *testing.B) {
	for b.Loop() {
		n := 0
		for range string(longB) {
			n++
		}
		intSink = n
	}
}

// 对照：存进变量。小字符串不逃逸时用栈上 tmpBuf，所以仍是 0 allocs，
// 但多了一次 memmove，比直接比较慢 7 倍
func BenchmarkNotZeroCopy(b *testing.B) {
	for b.Loop() {
		s := string(shortB)
		boolSink = s == "hello world"
	}
}

// ---------------------------------------------------------------------------
// ③ 拼接的五种写法
// ---------------------------------------------------------------------------

const n = 100

func BenchmarkConcatOperator(b *testing.B) {
	for b.Loop() {
		s := ""
		for i := range n {
			s += strconv.Itoa(i)
		}
		strSink = s
	}
}

func BenchmarkConcatSprintf(b *testing.B) {
	for b.Loop() {
		s := ""
		for i := range n {
			s = fmt.Sprintf("%s%d", s, i)
		}
		strSink = s
	}
}

func BenchmarkConcatBuilder(b *testing.B) {
	for b.Loop() {
		var sb strings.Builder
		for i := range n {
			sb.WriteString(strconv.Itoa(i))
		}
		strSink = sb.String()
	}
}

func BenchmarkConcatBuilderGrow(b *testing.B) {
	for b.Loop() {
		var sb strings.Builder
		sb.Grow(n * 3)
		for i := range n {
			sb.WriteString(strconv.Itoa(i))
		}
		strSink = sb.String()
	}
}

func BenchmarkConcatJoin(b *testing.B) {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = strconv.Itoa(i)
	}
	b.ResetTimer()
	for b.Loop() {
		strSink = strings.Join(parts, "")
	}
}

func BenchmarkConcatAppendBytes(b *testing.B) {
	for b.Loop() {
		buf := make([]byte, 0, n*3)
		for i := range n {
			buf = strconv.AppendInt(buf, int64(i), 10)
		}
		strSink = string(buf)
	}
}

// ---------------------------------------------------------------------------
// ④ 数字转字符串
// ---------------------------------------------------------------------------

func BenchmarkItoa(b *testing.B) {
	for b.Loop() {
		strSink = strconv.Itoa(1234567)
	}
}

func BenchmarkSprintfD(b *testing.B) {
	for b.Loop() {
		strSink = fmt.Sprintf("%d", 1234567)
	}
}

func BenchmarkAppendInt(b *testing.B) {
	buf := make([]byte, 0, 16)
	for b.Loop() {
		bytesSink = strconv.AppendInt(buf[:0], 1234567, 10)
	}
}

// ---------------------------------------------------------------------------
// ⑤ 计数与遍历
// ---------------------------------------------------------------------------

func BenchmarkLen(b *testing.B) {
	for b.Loop() {
		intSink = len(long)
	}
}

func BenchmarkRuneCount(b *testing.B) {
	for b.Loop() {
		intSink = utf8.RuneCountInString(long)
	}
}

func BenchmarkRuneCountViaRunes(b *testing.B) {
	for b.Loop() {
		intSink = len([]rune(long))
	}
}

// ---------------------------------------------------------------------------
// ⑥ 大小写无关比较
// ---------------------------------------------------------------------------

func BenchmarkEqualFold(b *testing.B) {
	for b.Loop() {
		boolSink = strings.EqualFold(short, "HELLO WORLD")
	}
}

func BenchmarkToLowerCompare(b *testing.B) {
	for b.Loop() {
		boolSink = strings.ToLower(short) == strings.ToLower("HELLO WORLD")
	}
}
