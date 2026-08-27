// Package memo 提供一个并发安全、支持取消的函数结果缓存（记忆化）。
//
// **练习 9.3：** 扩展Func类型和`(*Memo).Get`方法，支持调用方提供一个可选的done channel，
// 使其具备通过该channel来取消整个操作的能力（§8.9）。一个被取消了的Func的调用结果不应该被缓存。
//
// 实现要点（在 memo5 的 monitor goroutine 版本上扩展）：
//
//  1. Func 和 Get 都新增 done 参数；done 为 nil 表示永不取消（nil channel 永远阻塞）。
//  2. 同一个 key 的计算是共享的，所以不能让某一个调用方的取消影响其他等待者。
//     entry 用引用计数 n 记录仍在等待的调用方数量，只有 n 归零且 f 还没算完时，
//     才关闭 entry 自己的 done 来通知 f 放弃，并把该 entry 从缓存中删除。
//  3. 已经算出来的结果优先于取消：缓存命中永远不会返回 ErrCancelled。
package memo

import "errors"

// ErrCancelled 表示调用方在结果就绪前通过 done channel 取消了本次请求。
var ErrCancelled = errors.New("memo: request cancelled")

// Func 是被记忆化的函数类型。done 被关闭时，实现方应尽快放弃计算并返回错误。
type Func func(key string, done <-chan struct{}) (any, error)

type result struct {
	value any
	err   error
}

type entry struct {
	res   result
	ready chan struct{} // f 计算完成后关闭
	done  chan struct{} // 关闭表示放弃计算（所有等待者都取消了）
	n     int           // 仍在等待该 entry 的调用方数量，只由 server goroutine 访问
}

type request struct {
	key      string
	done     <-chan struct{} // 调用方的取消信号，可以为 nil
	response chan<- result
}

// completion 由 deliver goroutine 在结束时发给 server，用于维护引用计数。
type completion struct {
	key string
	e   *entry
}

type Memo struct {
	requests    chan request
	completions chan completion
}

func New(f Func) *Memo {
	m := &Memo{
		requests:    make(chan request),
		completions: make(chan completion),
	}
	go m.server(f)
	return m
}

// Get 返回 f(key) 的结果，重复的 key 会复用缓存。
// 若在结果就绪前 done 被关闭，Get 返回 ErrCancelled，且该次调用的结果不会被缓存。
func (m *Memo) Get(key string, done <-chan struct{}) (any, error) {
	response := make(chan result)
	select {
	case m.requests <- request{key: key, done: done, response: response}:
	case <-done:
		// 请求还没被 server 接收就被取消了。
		return nil, ErrCancelled
	}
	res := <-response
	return res.value, res.err
}

func (m *Memo) Close() { close(m.requests) }

func (m *Memo) server(f Func) {
	cache := make(map[string]*entry)
	requests := m.requests // Close 之后置为 nil，不再接收新请求
	outstanding := 0       // 在途的 deliver goroutine 数量

	for {
		select {
		case req, ok := <-requests:
			if !ok {
				requests = nil
				if outstanding == 0 {
					return
				}
				continue
			}
			e := cache[req.key]
			if e == nil {
				// 这个 key 的第一个请求。
				e = &entry{
					ready: make(chan struct{}),
					done:  make(chan struct{}),
				}
				cache[req.key] = e
				go e.call(f, req.key)
			}
			e.n++
			outstanding++
			go m.deliver(req, e)

		case c := <-m.completions:
			outstanding--
			c.e.n--
			if c.e.n == 0 {
				select {
				case <-c.e.ready:
					// f 已经算完，结果有效，继续留在缓存里。
				default:
					// 已经没人等这个结果了：通知 f 放弃，并且不缓存被取消的结果。
					close(c.e.done)
					if cache[c.key] == c.e {
						delete(cache, c.key)
					}
				}
			}
			if requests == nil && outstanding == 0 {
				return
			}
		}
	}
}

func (e *entry) call(f Func, key string) {
	e.res.value, e.res.err = f(key, e.done)
	close(e.ready) // 广播就绪条件
}

// deliver 等待结果就绪或调用方取消，然后把结果发回给调用方。
// 注意：completion 先于 response 发送，这样 Get 返回时 server 已经完成清理，
// 后续对同一个 key 的 Get 一定能看到"被取消的结果没有被缓存"。
func (m *Memo) deliver(req request, e *entry) {
	var res result
	select {
	case <-e.ready:
		res = e.res
	case <-req.done:
		// 两者可能同时就绪，此时优先返回已经算好的结果。
		select {
		case <-e.ready:
			res = e.res
		default:
			res = result{err: ErrCancelled}
		}
	}
	m.completions <- completion{key: req.key, e: e}
	req.response <- res
}
