package tst

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 被测的 handler
func rangeHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("r")
	lo, hi, err := ParseRange(q)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{"lo": lo, "hi": hi, "sum": Sum(lo, hi)})
}

// ---------------------------------------------------------------------------
// 1. httptest.NewRecorder：不起真的服务器，直接调 handler
// ---------------------------------------------------------------------------

func TestRangeHandler(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantCode int
		wantSum  int
	}{
		{"正常", "3-5", http.StatusOK, 12},
		{"单值", "7", http.StatusOK, 7},
		{"非法", "a", http.StatusBadRequest, 0},
		{"空", "", http.StatusBadRequest, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/range?r="+tc.query, nil)
			rec := httptest.NewRecorder()

			rangeHandler(rec, req)

			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tc.wantCode, rec.Body.String())
			}
			if tc.wantCode != http.StatusOK {
				return
			}
			var got map[string]int
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("解析响应失败: %v (body=%s)", err, rec.Body.String())
			}
			if got["sum"] != tc.wantSum {
				t.Errorf("sum = %d, want %d", got["sum"], tc.wantSum)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 2. httptest.NewServer：需要真的走一遍网络栈时（测客户端、中间件、超时）
// ---------------------------------------------------------------------------

func TestWithRealServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(rangeHandler))
	t.Cleanup(srv.Close) // 用 Cleanup 而不是 defer，辅助函数里也能注册

	resp, err := srv.Client().Get(srv.URL + "/range?r=1-10")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"sum":55`) {
		t.Errorf("body = %s", body)
	}
	t.Logf("server URL = %s（每次随机端口）", srv.URL)
}

// ---------------------------------------------------------------------------
// 3. 用 RoundTripper 打桩，测"调用外部 HTTP 服务"的代码
// ---------------------------------------------------------------------------

type stubTransport struct {
	status int
	body   string
	gotURL string
}

func (s *stubTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	s.gotURL = r.URL.String()
	return &http.Response{
		StatusCode: s.status,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     make(http.Header),
	}, nil
}

// 被测代码：依赖注入一个 *http.Client
func fetchSum(c *http.Client, url string) (int, error) {
	resp, err := c.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var out struct{ Sum int }
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}
	return out.Sum, nil
}

func TestFetchSumWithStub(t *testing.T) {
	st := &stubTransport{status: 200, body: `{"sum":42}`}
	client := &http.Client{Transport: st}

	got, err := fetchSum(client, "http://example.invalid/range?r=1-2")
	if err != nil {
		t.Fatal(err)
	}
	if got != 42 {
		t.Errorf("sum = %d, want 42", got)
	}
	if !strings.Contains(st.gotURL, "r=1-2") {
		t.Errorf("请求的 URL = %q", st.gotURL)
	}
	t.Log("→ 打桩 Transport 比起 httptest.NewServer 更快、更好控制错误分支")
}
