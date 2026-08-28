package tst

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// 1. 表驱动 + 子测试：Go 测试的默认形态
// ---------------------------------------------------------------------------

func TestParseRange(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantLo  int
		wantHi  int
		wantErr error // 用 errors.Is 比较，不比字符串
	}{
		{name: "单个数字", in: "5", wantLo: 5, wantHi: 5},
		{name: "正常区间", in: "3-7", wantLo: 3, wantHi: 7},
		{name: "带空格", in: " 3 - 7 ", wantLo: 3, wantHi: 7},
		{name: "同值区间", in: "4-4", wantLo: 4, wantHi: 4},
		{name: "空字符串", in: "", wantErr: ErrBadRange},
		{name: "非数字", in: "a", wantErr: ErrBadRange},
		{name: "lo 大于 hi", in: "9-2", wantErr: ErrBadRange},
		{name: "缺右边", in: "3-", wantErr: ErrBadRange},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lo, hi, err := ParseRange(tc.in)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("ParseRange(%q) err = %v, want errors.Is(..., %v)", tc.in, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRange(%q) unexpected err = %v", tc.in, err)
			}
			if lo != tc.wantLo || hi != tc.wantHi {
				t.Errorf("ParseRange(%q) = (%d, %d), want (%d, %d)", tc.in, lo, hi, tc.wantLo, tc.wantHi)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 2. t.Parallel：子测试并行
// ---------------------------------------------------------------------------

func TestParallel(t *testing.T) {
	// 1.22 之前这里必须写 tc := tc；1.22 起循环变量每轮新建，不需要了
	for _, in := range []string{"1", "2-3", "4-9"} {
		t.Run(in, func(t *testing.T) {
			t.Parallel() // 让这些子测试并行跑
			if _, _, err := ParseRange(in); err != nil {
				t.Errorf("ParseRange(%q): %v", in, err)
			}
		})
	}
	// ⚠️ 父测试的代码会在所有并行子测试**开始之前**跑完；
	//    要在并行子测试之后做事，得放在另一个 t.Run 里或用 t.Cleanup
}

// ---------------------------------------------------------------------------
// 3. t.Cleanup：比 defer 更适合测试
// ---------------------------------------------------------------------------

func newTestStore(t *testing.T) *Store {
	t.Helper() // 报错时显示调用者的行号，而不是这里的

	s := NewStore()
	t.Cleanup(func() { // 注册在 t 上，子测试结束时自动执行
		s.Close()
		t.Logf("store closed (cleanup for %s)", t.Name())
	})
	return s
}

func TestCleanup(t *testing.T) {
	s := newTestStore(t) // 辅助函数里注册的 Cleanup 也生效——这是它比 defer 强的地方

	if err := s.Put("k", "v"); err != nil {
		t.Fatal(err)
	}
	if v, ok := s.Get("k"); !ok || v != "v" {
		t.Errorf("Get(k) = %q, %v", v, ok)
	}

	t.Run("子测试有自己的 cleanup", func(t *testing.T) {
		s2 := newTestStore(t)
		_ = s2.Put("a", "b")
	}) // 这里 s2 已经被 Close

	if s.closed {
		t.Error("父测试的 store 不该被子测试的 cleanup 关掉")
	}
}

// ---------------------------------------------------------------------------
// 4. 环境隔离：t.Setenv / t.TempDir / t.Chdir
// ---------------------------------------------------------------------------

func TestEnvIsolation(t *testing.T) {
	// t.Setenv：测试结束自动还原；用了它的测试不能 t.Parallel（会 panic）
	t.Setenv("TST_MODE", "unit")
	if got := os.Getenv("TST_MODE"); got != "unit" {
		t.Errorf("TST_MODE = %q", got)
	}

	// t.TempDir：每个测试独立目录，结束自动删除
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(f, []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("TempDir = %s", dir)

	// t.Chdir（1.24+）：切工作目录，结束自动还原；同样不能和 Parallel 共用
	t.Chdir(dir)
	wd, _ := os.Getwd()
	if !strings.HasSuffix(wd, filepath.Base(dir)) {
		t.Errorf("Getwd = %q, want suffix %q", wd, filepath.Base(dir))
	}
}

// ---------------------------------------------------------------------------
// 5. t.Context（1.24+）：测试结束/超时时自动取消
// ---------------------------------------------------------------------------

func TestContext(t *testing.T) {
	ctx := t.Context() // 在 Cleanup 之前被 cancel

	select {
	case <-ctx.Done():
		t.Fatal("不该已经被取消")
	default:
	}

	if dl, ok := t.Deadline(); ok {
		t.Logf("本次测试的 deadline（来自 -timeout）: %v", dl)
	} else {
		t.Log("没有 deadline（-timeout=0）")
	}
}

// ---------------------------------------------------------------------------
// 6. Skip：按条件跳过
// ---------------------------------------------------------------------------

func TestSkipExamples(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过：-short 模式")
	}
	t.Run("需要网络", func(t *testing.T) {
		if os.Getenv("TST_NETWORK") == "" {
			t.Skip("跳过：需要 TST_NETWORK=1")
		}
	})
	t.Log("testing.Short() =", testing.Short())
}

// ---------------------------------------------------------------------------
// 7. 1.26 新增：t.Attr / t.ArtifactDir
// ---------------------------------------------------------------------------

func TestAttrAndArtifacts(t *testing.T) {
	// Attr：给测试打结构化标签，会出现在 test2json 输出里，方便 CI 聚合
	t.Attr("owner", "platform-team")
	t.Attr("ticket", "GO-1234")

	// ArtifactDir：存放测试产物（截图、pprof、失败时的输入）
	// 加 -artifacts 时落在输出目录，否则是临时目录
	dir := t.ArtifactDir()
	if err := os.WriteFile(filepath.Join(dir, "report.txt"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("ArtifactDir = %s", dir)
}
