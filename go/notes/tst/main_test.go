package tst

import (
	"fmt"
	"os"
	"testing"
)

// TestMain：整个测试二进制的入口，用来做**包级别**的 setup/teardown。
//
// 典型场景：起一个测试数据库容器、跑迁移、准备共享 fixture、
// 检查 goroutine 泄漏、设置全局 flag。
//
// 注意：
//   - 有 TestMain 时，测试不会自动跑，必须显式调用 m.Run()
//   - m.Run() 返回退出码，要用 os.Exit 传出去
//   - os.Exit 不执行 defer，所以清理逻辑要写在 Exit 之前（或者用一个内层函数）
//   - 一个包只能有一个 TestMain

func TestMain(m *testing.M) {
	// setup
	if err := setupPackage(); err != nil {
		fmt.Fprintln(os.Stderr, "setup failed:", err)
		os.Exit(1)
	}

	code := m.Run() // 跑本包所有 Test/Benchmark/Example/Fuzz

	// teardown（注意：这里不能用 defer，因为下面就 os.Exit 了）
	teardownPackage()

	os.Exit(code)
}

var pkgStore *Store

func setupPackage() error {
	pkgStore = NewStore()
	return pkgStore.Put("shared", "fixture")
}

func teardownPackage() {
	if pkgStore != nil {
		pkgStore.Close()
	}
}

func TestUsesSharedFixture(t *testing.T) {
	v, ok := pkgStore.Get("shared")
	if !ok || v != "fixture" {
		t.Errorf("shared fixture = %q, %v", v, ok)
	}
}
