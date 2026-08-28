// Package tst 是 notes/test.md 的配套代码：被测代码 + 各种测试写法。
//
//	go test ./tst                      跑单元测试
//	go test -v -run TestTable ./tst    看子测试树
//	go test -bench . -benchmem ./tst   跑 benchmark
//	go test -fuzz FuzzParseRange ./tst 跑模糊测试（Ctrl+C 停）
//	go test -race ./tst                竞态检测
//	go test -cover ./tst               覆盖率
package tst

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// 被测代码：一个"解析范围表达式"的小函数，足够产生边界情况
// ---------------------------------------------------------------------------

var ErrBadRange = errors.New("tst: bad range")

// ParseRange 解析 "3-7" 这样的表达式，返回闭区间 [lo, hi]。
// 单个数字 "5" 视为 [5,5]。
func ParseRange(s string) (lo, hi int, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, fmt.Errorf("%w: empty", ErrBadRange)
	}

	before, after, found := strings.Cut(s, "-")
	if !found {
		n, cerr := strconv.Atoi(s)
		if cerr != nil {
			return 0, 0, fmt.Errorf("%w: %q: %w", ErrBadRange, s, cerr)
		}
		return n, n, nil
	}

	lo, err = strconv.Atoi(strings.TrimSpace(before))
	if err != nil {
		return 0, 0, fmt.Errorf("%w: lo %q: %w", ErrBadRange, before, err)
	}
	hi, err = strconv.Atoi(strings.TrimSpace(after))
	if err != nil {
		return 0, 0, fmt.Errorf("%w: hi %q: %w", ErrBadRange, after, err)
	}
	if lo > hi {
		return 0, 0, fmt.Errorf("%w: %d > %d", ErrBadRange, lo, hi)
	}
	return lo, hi, nil
}

// Sum 把范围内的整数加起来（故意写成循环，给 benchmark 用）。
func Sum(lo, hi int) int {
	total := 0
	for i := lo; i <= hi; i++ {
		total += i
	}
	return total
}

// Store 是一个需要"清理"的依赖，用来演示 t.Cleanup。
type Store struct {
	data   map[string]string
	closed bool
}

func NewStore() *Store { return &Store{data: map[string]string{}} }

func (s *Store) Put(k, v string) error {
	if s.closed {
		return errors.New("tst: store closed")
	}
	s.data[k] = v
	return nil
}

func (s *Store) Get(k string) (string, bool) {
	v, ok := s.data[k]
	return v, ok
}

func (s *Store) Close() { s.closed = true }
