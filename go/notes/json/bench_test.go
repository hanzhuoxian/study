package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// go test -bench . -benchmem ./json
//
// 对应 notes/json.md 二（encoderCache / typeFields 缓存）、2.3（map key 排序）、五（流式）。
//
// 实测（go1.26.3 darwin/amd64, i5-1038NG7）：
//
//	BenchmarkMarshalStruct-8            402.7 ns/op    144 B/op     2 allocs/op
//	BenchmarkMarshalMap-8              1154   ns/op    368 B/op    10 allocs/op  ← key 提取+排序
//	BenchmarkMarshalToBuf-8             410.3 ns/op    144 B/op     2 allocs/op
//	BenchmarkEncoderReuse-8             365.8 ns/op     64 B/op     1 allocs/op  ← 复用 buffer
//	BenchmarkUnmarshalStruct-8         1917   ns/op    432 B/op    11 allocs/op
//	BenchmarkUnmarshalAny-8            1746   ns/op    792 B/op    23 allocs/op  ← 快一点但分配翻倍
//	BenchmarkUnmarshalRawMessage-8     1509   ns/op    400 B/op     9 allocs/op
//	BenchmarkReadAllThenUnmarshal-8  232113   ns/op  68000 B/op  1600 allocs/op
//	BenchmarkStreamDecoder-8         169569   ns/op  22096 B/op   611 allocs/op
//
// 两点和直觉不同：
//  1. 解到 any 并不比解到 struct 慢（省掉了反射字段匹配），但分配多一倍——
//     map[string]any 每个 key/value 都要装箱，内存和 GC 压力才是代价。
//  2. struct 的 Marshal 快在 typeFields/encoderFunc 全被缓存；map 每次都要重新
//     提取 key、排序，所以慢 3 倍。

type benchUser struct {
	Name  string   `json:"name"`
	Age   int      `json:"age"`
	Email string   `json:"email"`
	Tags  []string `json:"tags"`
}

var (
	benchVal = benchUser{"bob", 30, "bob@example.com", []string{"a", "b", "c"}}
	benchMap = map[string]any{
		"name": "bob", "age": 30, "email": "bob@example.com",
		"tags": []string{"a", "b", "c"},
	}
	benchJSON = []byte(`{"name":"bob","age":30,"email":"bob@example.com","tags":["a","b","c"]}`)

	bytesSink []byte
	errSink   error
)

// ---------------------------------------------------------------------------
// struct vs map：map 多了一次 key 提取 + 排序
// ---------------------------------------------------------------------------

func BenchmarkMarshalStruct(b *testing.B) {
	for b.Loop() {
		bytesSink, errSink = json.Marshal(benchVal)
	}
}

func BenchmarkMarshalMap(b *testing.B) {
	for b.Loop() {
		bytesSink, errSink = json.Marshal(benchMap)
	}
}

// ---------------------------------------------------------------------------
// Marshal vs Encoder：Encoder 复用 buffer，省掉每次返回 []byte 的分配
// ---------------------------------------------------------------------------

func BenchmarkMarshalToBuf(b *testing.B) {
	for b.Loop() {
		bytesSink, errSink = json.Marshal(benchVal)
	}
}

func BenchmarkEncoderReuse(b *testing.B) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for b.Loop() {
		buf.Reset()
		errSink = enc.Encode(benchVal)
	}
}

// ---------------------------------------------------------------------------
// Unmarshal：struct vs any（map[string]any）vs RawMessage 延迟解析
// ---------------------------------------------------------------------------

func BenchmarkUnmarshalStruct(b *testing.B) {
	for b.Loop() {
		var u benchUser
		errSink = json.Unmarshal(benchJSON, &u)
	}
}

func BenchmarkUnmarshalAny(b *testing.B) {
	for b.Loop() {
		var v any
		errSink = json.Unmarshal(benchJSON, &v)
	}
}

// 只需要外层的 type 时，内层用 RawMessage 完全不解析
func BenchmarkUnmarshalRawMessage(b *testing.B) {
	data := []byte(`{"type":"user","data":` + string(benchJSON) + `}`)
	for b.Loop() {
		var envelope struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		errSink = json.Unmarshal(data, &envelope)
	}
}

// ---------------------------------------------------------------------------
// 流式：Decoder 边读边解，不用先 io.ReadAll
// ---------------------------------------------------------------------------

var lines = strings.Repeat(string(benchJSON)+"\n", 100)

func BenchmarkReadAllThenUnmarshal(b *testing.B) {
	for b.Loop() {
		r := strings.NewReader(lines)
		for {
			// 模拟"必须拿到完整 []byte 才能 Unmarshal"的写法
			line, err := readLine(r)
			if err == io.EOF {
				break
			}
			var u benchUser
			errSink = json.Unmarshal(line, &u)
		}
	}
}

func BenchmarkStreamDecoder(b *testing.B) {
	for b.Loop() {
		dec := json.NewDecoder(strings.NewReader(lines))
		for {
			var u benchUser
			if err := dec.Decode(&u); err == io.EOF {
				break
			} else if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func readLine(r *strings.Reader) ([]byte, error) {
	var buf []byte
	for {
		c, err := r.ReadByte()
		if err != nil {
			if len(buf) == 0 {
				return nil, io.EOF
			}
			return buf, nil
		}
		if c == '\n' {
			return buf, nil
		}
		buf = append(buf, c)
	}
}
