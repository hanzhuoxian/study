// error 示例：对应 notes/error.md
// 运行：go run ./errs
// 目录叫 errs 而不是 error，避免和内置类型名混淆
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

// note 打印一行说明文字。用它而不是 fmt.Println 是因为文案里含 %w/%v 这类动词，
// 直接传给 fmt.Println 会被 go vet 判为"疑似格式化指令"。
func note(s string) { fmt.Println(s) }

func main() {
	basicInterface()
	basicSentinel()
	basicCustomType()
	basicWrap()
	basicIsAs()
	basicAsType()
	basicJoin()
	basicMultiWrap()

	principleUnwrapTree()
	principleIsAsCustom()

	trapNilInterface()
	trapEqualCompare()
	trapAsNonPointer()
	trapWrapOrNot()
	trapErrorStringConcat()
	trapPanicVsError()
	trapDeferErr()
	trapSentinelLeak()
}

// ---------------------------------------------------------------------------
// 1.1 error 就是一个接口
// ---------------------------------------------------------------------------

func basicInterface() {
	section("1.1 error 接口")

	fmt.Println("type error interface { Error() string }   // 就这一行，在 builtin 里")

	err := errors.New("something failed")
	fmt.Printf("errors.New 返回 %T（指针！所以两个同文本的 New 不相等）\n", err)
	fmt.Println("errors.New(\"x\") == errors.New(\"x\") ?", errors.New("x") == errors.New("x"))

	// fmt.Errorf 不带 %w 时等价于 errors.New(格式化结果)
	e2 := fmt.Errorf("id=%d not found", 42)
	fmt.Printf("fmt.Errorf 不带 %%w 返回 %T\n", e2)
	fmt.Println("Unwrap 结果:", errors.Unwrap(e2))
}

// ---------------------------------------------------------------------------
// 1.2 哨兵错误（sentinel error）
// ---------------------------------------------------------------------------

var (
	ErrNotFound   = errors.New("user: not found")
	ErrPermission = errors.New("user: permission denied")
)

func findUser(id int) error {
	switch id {
	case 0:
		return ErrNotFound
	case 1:
		return fmt.Errorf("findUser(%d): %w", id, ErrPermission) // 包一层
	}
	return nil
}

func basicSentinel() {
	section("1.2 哨兵错误")

	for _, id := range []int{0, 1, 2} {
		err := findUser(id)
		fmt.Printf("  id=%d err=%-45v Is(ErrNotFound)=%-5v Is(ErrPermission)=%v\n",
			id, err, errors.Is(err, ErrNotFound), errors.Is(err, ErrPermission))
	}
	fmt.Println("→ 标准库的哨兵：io.EOF、sql.ErrNoRows、fs.ErrNotExist、context.Canceled、errors.ErrUnsupported")
	fmt.Println("→ 哨兵是包的公开 API，一旦导出就不能改；不想承诺就别导出（见 3.8）")
}

// ---------------------------------------------------------------------------
// 1.3 自定义错误类型
// ---------------------------------------------------------------------------

type ValidationError struct {
	Field string
	Value any
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: field %q got invalid value %v", e.Field, e.Value)
}

// 带 Unwrap 的自定义类型：既能带上下文，又能保留底层错误
type QueryError struct {
	Query string
	Err   error
}

func (e *QueryError) Error() string { return fmt.Sprintf("query %q: %v", e.Query, e.Err) }
func (e *QueryError) Unwrap() error { return e.Err }

func basicCustomType() {
	section("1.3 自定义错误类型")

	var err error = &ValidationError{Field: "age", Value: -1}
	fmt.Println(" ", err)

	var ve *ValidationError
	if errors.As(err, &ve) {
		fmt.Printf("  As 拿到结构化信息: Field=%s Value=%v\n", ve.Field, ve.Value)
	}

	err = &QueryError{Query: "SELECT 1", Err: sql.ErrNoRows}
	fmt.Println(" ", err)
	fmt.Println("  Is(sql.ErrNoRows) =", errors.Is(err, sql.ErrNoRows), "（靠 Unwrap 穿透）")

	fmt.Println("→ 什么时候用类型而不是哨兵：调用方需要拿到结构化字段（字段名、行号、状态码）")
	fmt.Println("→ 指针接收者是惯例：值接收者会让 As 的目标类型变得别扭，也容易出现 3.1 的坑")
}

// ---------------------------------------------------------------------------
// 1.4 包装：%w 与 Unwrap
// ---------------------------------------------------------------------------

func layer3() error { return io.ErrUnexpectedEOF }
func layer2() error {
	if err := layer3(); err != nil {
		return fmt.Errorf("layer2: read header: %w", err)
	}
	return nil
}
func layer1() error {
	if err := layer2(); err != nil {
		return fmt.Errorf("layer1: parse file: %w", err)
	}
	return nil
}

func basicWrap() {
	section("1.4 包装与 Unwrap 链")

	err := layer1()
	fmt.Println("  最终错误:", err)

	fmt.Println("  Unwrap 链:")
	for i, e := 0, err; e != nil; i, e = i+1, errors.Unwrap(e) {
		fmt.Printf("    [%d] %-14T %v\n", i, e, e)
	}
	fmt.Println("  Is(io.ErrUnexpectedEOF) =", errors.Is(err, io.ErrUnexpectedEOF))

	note("→ %w 和 %v 的唯一区别：%w 保留可编程的关系（Is/As 能穿透），%v 只留字符串")
	fmt.Println("→ 每层加的信息应该是'这一层特有的上下文'，不要重复下层已有的内容")
}

// ---------------------------------------------------------------------------
// 1.5 Is / As
// ---------------------------------------------------------------------------

func basicIsAs() {
	section("1.5 Is 与 As")

	_, err := os.Open("/definitely/not/exist")
	fmt.Println("  os.Open 错误:      ", err)
	fmt.Println("  Is(fs.ErrNotExist):", errors.Is(err, fs.ErrNotExist))
	fmt.Println("  os.IsNotExist:     ", os.IsNotExist(err), "（老 API，不穿透 wrap，别再用）")

	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		fmt.Printf("  As(*fs.PathError): Op=%s Path=%s Err=%v\n", pathErr.Op, pathErr.Path, pathErr.Err)
	}

	// Is 比较的是"值"，As 比较的是"类型"
	fmt.Println("  规则: Is 找值相等（或自定义 Is 方法为真），As 找类型可赋值（或自定义 As 方法为真）")

	_, numErr := strconv.Atoi("abc")
	var ne *strconv.NumError
	fmt.Println("  strconv 错误 As:  ", errors.As(numErr, &ne), ne.Func, ne.Num, ne.Err)
}

// ---------------------------------------------------------------------------
// 1.6 AsType（Go 1.26 新增的泛型版 As）
// ---------------------------------------------------------------------------

func basicAsType() {
	section("1.6 errors.AsType[E]（1.26+）")

	_, err := os.Open("/definitely/not/exist")

	// 老写法：先声明变量，再传指针，靠 bool 判断
	var pathErr *fs.PathError
	ok1 := errors.As(err, &pathErr)

	// 新写法：一行，类型安全，不需要预声明
	pe, ok2 := errors.AsType[*fs.PathError](err)

	fmt.Printf("  errors.As:              ok=%v path=%s\n", ok1, pathErr.Path)
	fmt.Printf("  errors.AsType[*PathError]: ok=%v path=%s\n", ok2, pe.Path)
	fmt.Println("→ 签名: func AsType[E error](err error) (E, bool)")
	fmt.Println("→ 好处：不再有 'As 的第二个参数必须是指向 error 类型的指针' 这个运行时 panic")
}

// ---------------------------------------------------------------------------
// 1.7 Join：多个错误合成一个（1.20+）
// ---------------------------------------------------------------------------

func validate(age int, name string) error {
	var errs []error
	if age < 0 {
		errs = append(errs, &ValidationError{Field: "age", Value: age})
	}
	if name == "" {
		errs = append(errs, &ValidationError{Field: "name", Value: name})
	}
	return errors.Join(errs...) // 全 nil 时返回 nil
}

func basicJoin() {
	section("1.7 errors.Join（1.20+）")

	err := validate(-1, "")
	fmt.Println("  合并后（Error() 用 \\n 连接）:")
	for _, line := range strings.Split(err.Error(), "\n") {
		fmt.Println("    " + line)
	}

	fmt.Println("  Is 仍然能穿透:", errors.Is(err, err))
	var ve *ValidationError
	fmt.Println("  As 拿到第一个匹配:", errors.As(err, &ve), ve.Field)

	fmt.Println("  Join(nil, nil) =", errors.Join(nil, nil), "（全 nil 返回 nil）")
	fmt.Println("  Unwrap() error 拿不到:", errors.Unwrap(err), "（joinError 只实现了 Unwrap() []error）")

	if u, ok := err.(interface{ Unwrap() []error }); ok {
		fmt.Println("  用 Unwrap() []error 拿到", len(u.Unwrap()), "个子错误")
	}

	fmt.Println("→ 典型场景：表单校验、批量任务、多个 defer Close 的错误汇总")
}

// ---------------------------------------------------------------------------
// 1.8 一个 Errorf 里多个 %w（1.20+）
// ---------------------------------------------------------------------------

func basicMultiWrap() {
	section("1.8 多个 %w")

	err := fmt.Errorf("both failed: %w and %w", ErrNotFound, ErrPermission)
	fmt.Println(" ", err)
	fmt.Println("  Is(ErrNotFound)  =", errors.Is(err, ErrNotFound))
	fmt.Println("  Is(ErrPermission)=", errors.Is(err, ErrPermission))
	fmt.Printf("  底层类型 %T（fmt.wrapErrors，实现 Unwrap() []error）\n", err)
}

// ---------------------------------------------------------------------------
// 2.1 Unwrap 树与遍历顺序
// ---------------------------------------------------------------------------

func principleUnwrapTree() {
	section("2.1 Unwrap 是树，不是链")

	leaf1 := errors.New("leaf1")
	leaf2 := errors.New("leaf2")
	branch := errors.Join(leaf1, leaf2)
	root := fmt.Errorf("root: %w", branch)

	fmt.Println("  结构: root -> Join(leaf1, leaf2)")
	fmt.Println("  Is(leaf1) =", errors.Is(root, leaf1), " Is(leaf2) =", errors.Is(root, leaf2))
	fmt.Println("→ 两种 Unwrap 方法：")
	fmt.Println("    Unwrap() error    单个子错误（fmt.Errorf 一个 %w、自定义类型）")
	fmt.Println("    Unwrap() []error  多个子错误（errors.Join、多个 %w）")
	fmt.Println("→ Is/As 的遍历是先序深度优先（pre-order, depth-first）")
	fmt.Println("→ 注意 errors.Unwrap() 函数只认 Unwrap() error，对 Join 返回 nil")
}

// ---------------------------------------------------------------------------
// 2.2 自定义 Is / As 方法
// ---------------------------------------------------------------------------

type HTTPError struct {
	Code int
}

func (e *HTTPError) Error() string { return fmt.Sprintf("http status %d", e.Code) }

// 自定义 Is：让所有 5xx 都匹配 ErrServerSide
func (e *HTTPError) Is(target error) bool {
	if target == ErrServerSide {
		return e.Code >= 500
	}
	return false
}

var ErrServerSide = errors.New("server side error")

func principleIsAsCustom() {
	section("2.2 自定义 Is 方法")

	for _, code := range []int{404, 500, 503} {
		err := error(&HTTPError{Code: code})
		fmt.Printf("  code=%d Is(ErrServerSide)=%v\n", code, errors.Is(err, ErrServerSide))
	}
	fmt.Println("→ 实现 Is(target error) bool 可以表达'一类错误'的语义")
	fmt.Println("→ 也可以实现 As(target any) bool 自己做类型转换（标准库里 fs.PathError 没这么做，")
	fmt.Println("  但 net.OpError 之类的复合错误常用）")
}

// ---------------------------------------------------------------------------
// 3.1 nil 接口陷阱：最经典的 Go 面试题
// ---------------------------------------------------------------------------

type MyErr struct{}

func (e *MyErr) Error() string { return "my error" }

// ✗ 返回类型是具体指针类型
func badReturn(fail bool) *MyErr {
	if fail {
		return &MyErr{}
	}
	return nil // 一个 nil 的 *MyErr
}

// ✓ 返回 error 接口
func goodReturn(fail bool) error {
	if fail {
		return &MyErr{}
	}
	return nil // 真正的 nil error
}

func trapNilInterface() {
	section("3.1 nil 接口陷阱")

	var err error = badReturn(false) // 把 nil *MyErr 装进接口
	fmt.Printf("  err = badReturn(false) -> err == nil ? %v （类型 %T）\n", err == nil, err)
	fmt.Printf("  err = goodReturn(false) -> err == nil ? %v\n", goodReturn(false) == nil)

	fmt.Println("→ 接口值 = (type, data) 两个字，只有两者都为 nil 接口才等于 nil；")
	fmt.Println("  这里 type=*main.MyErr、data=nil，所以 err != nil（interface.md 2.1）")
	fmt.Println("→ 铁律：函数返回错误一律声明为 error，绝不返回具体的错误指针类型")
	fmt.Println("→ 检测：go vet 的 nilness 分析、staticcheck SA4023 能抓到一部分")
}

// ---------------------------------------------------------------------------
// 3.2 用 == 比较错误
// ---------------------------------------------------------------------------

func trapEqualCompare() {
	section("3.2 用 == 比较错误")

	wrapped := fmt.Errorf("read config: %w", os.ErrNotExist)
	fmt.Println("  wrapped == os.ErrNotExist        ?", wrapped == os.ErrNotExist, "← 包过就不相等了")
	fmt.Println("  errors.Is(wrapped, os.ErrNotExist)?", errors.Is(wrapped, os.ErrNotExist))

	fmt.Println("→ 唯一还能用 == 的地方：io.EOF（约定俗成，标准库承诺不包装它）")
	fmt.Println("   即便如此也建议写 errors.Is(err, io.EOF)")
	fmt.Println("→ 比较错误字符串（err.Error() == \"...\"）是最糟的写法：文案随时会改")
}

// ---------------------------------------------------------------------------
// 3.3 errors.As 的参数要求
// ---------------------------------------------------------------------------

func trapAsNonPointer() {
	section("3.3 errors.As 的参数陷阱")

	err := fmt.Errorf("wrap: %w", &ValidationError{Field: "x"})

	func() {
		defer func() { fmt.Println("  panic:", recover()) }()
		var ve *ValidationError
		asFn := errors.As // 经一次函数值间接调用，绕开 go vet 的静态检查
		asFn(err, ve)     // ✗ 传的不是 &ve
	}()

	var ve *ValidationError
	fmt.Println("  正确写法 errors.As(err, &ve):", errors.As(err, &ve))
	pe, ok := errors.AsType[*ValidationError](err)
	fmt.Println("  1.26 起更好: errors.AsType[*ValidationError](err) ->", ok, pe.Field)
	fmt.Println("→ As 的第二个参数是 any，参数类型错误只能在运行时 panic；AsType 把它变成了编译期检查")
}

// ---------------------------------------------------------------------------
// 3.4 该不该包装
// ---------------------------------------------------------------------------

func trapWrapOrNot() {
	section("3.4 包装 vs 不包装")

	note("  用 %w（暴露底层错误，成为你的 API 契约）：")
	fmt.Println("    · 调用方确实需要 Is/As 判断底层原因")
	fmt.Println("    · 同一个模块内部的层级传递")
	note("  用 %v（只留文字，切断依赖）：")
	fmt.Println("    · 底层是实现细节（今天用 MySQL 明天换 Redis）")
	fmt.Println("    · 不想让调用方依赖第三方库的错误类型")
	fmt.Println("  直接 return err（不加上下文）：")
	fmt.Println("    · 这一层没有任何有价值的新信息可加")
	note("→ 反模式：每一层都 fmt.Errorf(\"xxx: %w\") 结果日志里出现十层重复前缀")
	fmt.Println("→ 也别在包装时重复动词：\"failed to open file: failed to open: no such file\"")
}

// ---------------------------------------------------------------------------
// 3.5 错误文案规范
// ---------------------------------------------------------------------------

func trapErrorStringConcat() {
	section("3.5 错误文案")

	fmt.Println("  ✓ 小写开头、不带标点：errors.New(\"user: not found\")")
	fmt.Println("  ✗ 大写开头/句号:      errors.New(\"User not found.\")")
	fmt.Println("  原因：错误常被嵌进更长的句子里（\"read config: user: not found\"）")
	fmt.Println("  例外：以专有名词/缩写开头时保留原样（\"HTTP request failed\"、\"TLS handshake\"）")
	fmt.Println("  惯例：加上包名/操作名做前缀，形成天然的调用链（\"pkg: op: detail\"）")
	fmt.Println("  go lint 规则：ST1005（error strings should not be capitalized）")
}

// ---------------------------------------------------------------------------
// 3.6 panic vs error
// ---------------------------------------------------------------------------

func trapPanicVsError() {
	section("3.6 panic vs error")

	fmt.Println("  用 error：一切可预期的失败——IO、网络、解析、校验、业务规则")
	fmt.Println("  用 panic：程序员的 bug，继续执行只会更糟——")
	fmt.Println("    · 不可能发生的分支（default: panic(\"unreachable\")）")
	fmt.Println("    · 初始化阶段的致命配置错误（MustCompile、init 里）")
	fmt.Println("    · 违反不变量（内部状态自相矛盾）")
	fmt.Println("  库的边界要 recover：HTTP handler、goroutine 入口、插件调用点")
	fmt.Println("  ⚠️ recover 只在直接 defer 的函数里有效（func.md 3.4）")
	fmt.Println("  ⚠️ 别用 panic/recover 做控制流：跨函数抛错误在 Go 里是反模式")

	// 库里的标准做法：内部用 panic 简化代码，边界 recover 成 error
	fmt.Println("  示例 safeParse:", must(func() (int, error) { return strconv.Atoi("42") }))
	if _, err := safeDivide(1, 0); err != nil {
		fmt.Println("  safeDivide(1, 0) 把 panic 转成了 error:", err)
	}
}

func must[T any](f func() (T, error)) T {
	v, err := f()
	if err != nil {
		panic(err)
	}
	return v
}

func safeDivide(a, b int) (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("safeDivide: recovered: %v", r)
		}
	}()
	return a / b, nil
}

// ---------------------------------------------------------------------------
// 3.7 defer 里的错误
// ---------------------------------------------------------------------------

// ✗ Close 的错误被丢掉了（写文件时可能丢数据）
func writeBad(name string, data []byte) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close() // 错误被忽略
	_, err = f.Write(data)
	return err
}

// ✓ 用命名返回值 + errors.Join 汇总
func writeGood(name string, data []byte) (err error) {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, f.Close()) // 两个错误都保留
	}()
	_, err = f.Write(data)
	return err
}

func trapDeferErr() {
	section("3.7 defer 里的错误")

	tmp := os.TempDir() + "/notes_errs_demo.txt"
	fmt.Println("  writeBad :", writeBad(tmp, []byte("hi")), "（Close 的错误被吞了）")
	fmt.Println("  writeGood:", writeGood(tmp, []byte("hi")), "（errors.Join 汇总）")
	_ = os.Remove(tmp)

	fmt.Println("→ 只读场景 defer f.Close() 忽略错误没问题；写场景必须检查（数据可能还在缓冲区）")
	fmt.Println("→ errcheck / go vet 的 lostcancel 之类检查能帮忙发现被忽略的错误")
}

// ---------------------------------------------------------------------------
// 3.8 哨兵错误是 API 承诺
// ---------------------------------------------------------------------------

func trapSentinelLeak() {
	section("3.8 哨兵/包装是 API 契约")

	note("  一旦你 return 了 %w 包装的第三方错误，调用方就可能写：")
	fmt.Println("    errors.Is(err, pq.ErrSSLNotSupported)")
	fmt.Println("  于是你再也不能换掉那个数据库驱动——这是隐式的 API 泄漏")
	note("→ 对外的包：只暴露自己定义的哨兵/错误类型，内部错误用 %v 转成文字")
	fmt.Println("→ 需要分类时定义自己的层级：ErrTimeout / ErrConflict / ErrInvalidInput，")
	fmt.Println("  内部用自定义 Is 方法把底层错误映射上去（见 2.2）")
	fmt.Println("→ errors.ErrUnsupported（1.21+）是标准库给的通用哨兵：表示'这个操作不支持'")
	fmt.Println("   例：", fmt.Errorf("myfs: chmod: %w", errors.ErrUnsupported))
}
