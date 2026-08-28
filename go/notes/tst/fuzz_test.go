package tst

import (
	"errors"
	"strings"
	"testing"
)

// 模糊测试（1.18+）
//
//	go test -run FuzzParseRange ./tst          只跑种子语料（当普通测试）
//	go test -fuzz FuzzParseRange ./tst         真正开始 fuzz（Ctrl+C 停）
//	go test -fuzz FuzzParseRange -fuzztime 30s ./tst
//
// 崩溃输入会被写到 testdata/fuzz/FuzzParseRange/ 下，之后每次 go test 都会当种子跑，
// 也就是自动变成回归测试。

func FuzzParseRange(f *testing.F) {
	// 种子语料：覆盖已知的有意思的输入
	for _, seed := range []string{"", "5", "3-7", " 3 - 7 ", "9-2", "a", "-", "1-2-3", "+5", "０-１"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, s string) {
		lo, hi, err := ParseRange(s)

		// fuzz 测试断言的是**不变量**，不是具体值
		if err != nil {
			// 不变量 1：所有错误都必须能被 errors.Is(ErrBadRange) 识别
			if !errors.Is(err, ErrBadRange) {
				t.Fatalf("ParseRange(%q) 返回了未分类的错误: %v", s, err)
			}
			// 不变量 2：出错时返回值必须是零值
			if lo != 0 || hi != 0 {
				t.Fatalf("ParseRange(%q) 出错却返回了 (%d, %d)", s, lo, hi)
			}
			return
		}

		// 不变量 3：成功时必须 lo <= hi
		if lo > hi {
			t.Fatalf("ParseRange(%q) = (%d, %d)，违反 lo <= hi", s, lo, hi)
		}
		// 不变量 4：不 panic（f.Fuzz 里任何 panic 都算失败）
		_ = Sum(lo, min(hi, lo+1000))
	})
}

// 另一个例子：往返一致性（round-trip），fuzz 最擅长发现这类 bug
func FuzzRoundTrip(f *testing.F) {
	f.Add(3, 7)
	f.Add(0, 0)
	f.Add(-5, -1)

	f.Fuzz(func(t *testing.T, lo, hi int) {
		if lo > hi {
			lo, hi = hi, lo
		}
		s := formatRange(lo, hi)
		gotLo, gotHi, err := ParseRange(s)
		if err != nil {
			// 负数会被 "-" 分隔符搞混，这是 ParseRange 的已知局限，跳过
			if strings.Contains(s, "--") || lo < 0 {
				t.Skip()
			}
			t.Fatalf("ParseRange(%q) 解析自己格式化的结果失败: %v", s, err)
		}
		if gotLo != lo || gotHi != hi {
			t.Fatalf("往返不一致: (%d,%d) -> %q -> (%d,%d)", lo, hi, s, gotLo, gotHi)
		}
	})
}

func formatRange(lo, hi int) string {
	if lo == hi {
		return itoa(lo)
	}
	return itoa(lo) + "-" + itoa(hi)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [24]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
