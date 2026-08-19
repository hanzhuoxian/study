package main

// **练习 7.9：** 使用html/template包（§4.6）替代printTracks将tracks展示成一个HTML表格。
// 将这个解决方案用在前一个练习中，让每次点击一个列的头部产生一个HTTP请求来排序这个表格。

import (
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"
)

type Track struct {
	Title  string
	Artist string
	Album  string
	Year   int
	Length time.Duration
}

// compare 三态比较：x<y 返回负数，x==y 返回 0，x>y 返回正数。
// 多重排序需要识别"相等"才能决定是否继续比较下一个排序键。
type compare func(x, y *Track) int

// columnOrder 表头的展示顺序，map 遍历无序所以单独用切片固定下来。
var columnOrder = []string{"Title", "Artist", "Album", "Year", "Length"}

var columns = map[string]compare{
	"Title":  func(x, y *Track) int { return cmpString(x.Title, y.Title) },
	"Artist": func(x, y *Track) int { return cmpString(x.Artist, y.Artist) },
	"Album":  func(x, y *Track) int { return cmpString(x.Album, y.Album) },
	"Year":   func(x, y *Track) int { return cmpInt(x.Year, y.Year) },
	"Length": func(x, y *Track) int { return cmpInt(int(x.Length), int(y.Length)) },
}

func cmpString(x, y string) int {
	switch {
	case x < y:
		return -1
	case x > y:
		return 1
	}
	return 0
}

func cmpInt(x, y int) int { return x - y }

// sortKey 一个排序键：哪一列，升序还是降序。
type sortKey struct {
	column string
	desc   bool
}

// maxKeys 保留的历史排序键数量上限。
const maxKeys = 3

// MultiSorter 有状态的多重排序表格，keys[0] 为主排序键。
// 它同时被多个 HTTP 请求访问，所有读写都由 mu 保护。
type MultiSorter struct {
	mu     sync.Mutex
	tracks []*Track
	keys   []sortKey
}

func (m *MultiSorter) Len() int { return len(m.tracks) }

func (m *MultiSorter) Swap(i, j int) {
	m.tracks[i], m.tracks[j] = m.tracks[j], m.tracks[i]
}

// Less 按优先级依次比较各排序键，第一个不相等的键决定顺序。
func (m *MultiSorter) Less(i, j int) bool {
	x, y := m.tracks[i], m.tracks[j]
	for _, key := range m.keys {
		c := columns[key.column](x, y)
		if c == 0 {
			continue
		}
		if key.desc {
			return c > 0
		}
		return c < 0
	}
	return false
}

// Click 处理一次列头点击：该列成为新的主排序键；
// 若它已是主排序键则切换升/降序；若在历史中的其他位置则先摘除再提到最前。
func (m *MultiSorter) Click(column string) error {
	if _, ok := columns[column]; !ok {
		return fmt.Errorf("unknown column %q", column)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := sortKey{column: column}
	if len(m.keys) > 0 && m.keys[0].column == column {
		key.desc = !m.keys[0].desc
	}

	rest := m.keys[:0:0] // 不复用底层数组，避免覆盖尚未读到的元素
	for _, k := range m.keys {
		if k.column != column {
			rest = append(rest, k)
		}
	}

	m.keys = append([]sortKey{key}, rest...)
	if len(m.keys) > maxKeys {
		m.keys = m.keys[:maxKeys]
	}

	sort.Sort(m)
	return nil
}

// ---------- 视图模型 ----------

// columnView 一列表头的渲染数据。
type columnView struct {
	Name  string
	Arrow string // 升序 ▲ / 降序 ▼ / 未参与排序则为空
	Rank  int    // 该列在排序键中的优先级，1 为主键，0 表示未参与
}

type tableView struct {
	Columns []columnView
	Tracks  []*Track
}

// view 生成当前状态的一份快照供模板渲染，避免模板执行期间数据被并发修改。
func (m *MultiSorter) view() tableView {
	m.mu.Lock()
	defer m.mu.Unlock()

	cols := make([]columnView, 0, len(columnOrder))
	for _, name := range columnOrder {
		col := columnView{Name: name}
		for i, key := range m.keys {
			if key.column == name {
				col.Rank = i + 1
				col.Arrow = "▲"
				if key.desc {
					col.Arrow = "▼"
				}
				break
			}
		}
		cols = append(cols, col)
	}

	tracks := make([]*Track, len(m.tracks))
	copy(tracks, m.tracks)

	return tableView{Columns: cols, Tracks: tracks}
}

// ---------- HTML 模板 ----------

// 模板内容在编译期嵌入二进制，这样在任意目录下 go run 都能找到它。
// 表头是链接，点击产生一次 GET /?sort=<列名> 请求来重新排序。
//
//go:embed tracks.html
var tracksHTML string

var tableTemplate = template.Must(template.New("tracks").Parse(tracksHTML))

// ---------- HTTP ----------

func main() {
	addr := flag.String("addr", "localhost:8000", "HTTP service address")
	flag.Parse()

	table := &MultiSorter{tracks: newTracks()}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if col := r.URL.Query().Get("sort"); col != "" {
			if err := table.Click(col); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tableTemplate.Execute(w, table.view()); err != nil {
			log.Print(err)
		}
	})

	log.Printf("listening on http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func newTracks() []*Track {
	return []*Track{
		{"Go", "Delilah", "From the Roots Up", 2012, length("3m38s")},
		{"Go", "Moby", "Moby", 1992, length("3m37s")},
		{"Go", "Moby", "Moby Remixed", 1993, length("3m41s")},
		{"Go Ahead", "Alicia Keys", "As I Am", 2007, length("4m36s")},
		{"Go Ahead", "Alicia Keys", "Unplugged", 2005, length("4m58s")},
		{"Ready 2 Go", "Martin Solveig", "Smash", 2011, length("4m24s")},
	}
}

func length(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(s)
	}
	return d
}
