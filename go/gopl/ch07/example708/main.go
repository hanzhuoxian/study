package main

// **练习 7.8：** 很多图形界面提供了一个有状态的多重排序表格插件：主要的排序键是最近一次点击过列头的列，
// 第二个排序键是第二最近点击过列头的列，等等。定义一个sort.Interface的实现用在这样的表格中。
// 比较这个实现方式和重复使用sort.Stable来排序的方式。

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"text/tabwriter"
	"time"
)

type Track struct {
	Title  string
	Artist string
	Album  string
	Year   int
	Length time.Duration
}

// compare 返回三态结果：x<y 返回负数，x==y 返回 0，x>y 返回正数。
// 多重排序必须能区分"相等"，才能决定是否继续比较下一个排序键，
// 所以这里不能用 sort.Interface 那种只返回 bool 的 less 函数。
type compare func(x, y *Track) int

// columns 表格支持的所有列，列头点击时按名字查找比较函数。
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

func cmpInt(x, y int) int {
	return x - y
}

// sortKey 一个排序键：哪一列，以及升序还是降序。
type sortKey struct {
	column string
	desc   bool
}

// maxKeys 保留的历史排序键数量上限，避免点击历史无限增长。
const maxKeys = 3

// MultiSorter 有状态的多重排序表格。
// keys[0] 是主排序键（最近点击的列），keys[1] 是次排序键，依此类推。
type MultiSorter struct {
	tracks []*Track
	keys   []sortKey
}

func (m *MultiSorter) Len() int { return len(m.tracks) }

func (m *MultiSorter) Swap(i, j int) {
	m.tracks[i], m.tracks[j] = m.tracks[j], m.tracks[i]
}

// Less 按优先级依次比较各个排序键，第一个不相等的键决定顺序；
// 所有键都相等时返回 false（视为等价，保持原有相对次序由排序算法决定）。
func (m *MultiSorter) Less(i, j int) bool {
	x, y := m.tracks[i], m.tracks[j]
	for _, key := range m.keys {
		c := columns[key.column](x, y)
		if c == 0 {
			continue // 该键相等，交给下一个键判断
		}
		if key.desc {
			return c > 0
		}
		return c < 0
	}
	return false
}

// Click 模拟用户点击列头：
// 该列成为新的主排序键；如果它已经是主排序键，则切换升/降序；
// 如果它在历史中的其他位置，先摘掉再提升到最前，避免同一列出现两次。
func (m *MultiSorter) Click(column string) {
	if _, ok := columns[column]; !ok {
		panic("unknown column: " + column)
	}

	key := sortKey{column: column}
	if len(m.keys) > 0 && m.keys[0].column == column {
		key.desc = !m.keys[0].desc // 再次点击当前主键：反转方向
	}

	rest := m.keys[:0:0] // 不复用底层数组，防止覆盖 m.keys 元素
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
}

// ---------- 对照实现：重复使用 sort.Stable ----------

// stableSorter 用单个排序键实现 sort.Interface，供 sort.Stable 使用。
type stableSorter struct {
	tracks []*Track
	key    sortKey
}

func (s stableSorter) Len() int      { return len(s.tracks) }
func (s stableSorter) Swap(i, j int) { s.tracks[i], s.tracks[j] = s.tracks[j], s.tracks[i] }
func (s stableSorter) Less(i, j int) bool {
	c := columns[s.key.column](s.tracks[i], s.tracks[j])
	if s.key.desc {
		return c > 0
	}
	return c < 0
}

// sortByStable 用重复调用 sort.Stable 达到同样的多重排序效果。
// 关键在于顺序要反过来：从优先级最低的键排起，最后再排主键。
// sort.Stable 保证相等元素的相对次序不变，所以先前排序的结果自然成为并列时的后备顺序。
func sortByStable(tracks []*Track, keys []sortKey) {
	for _, key := range slices.Backward(keys) {
		sort.Stable(stableSorter{tracks: tracks, key: key})
	}
}

// ---------- 演示 ----------

func main() {
	table := &MultiSorter{tracks: newTracks()}

	// 依次点击 Year、Artist、Title：
	// 主键 Title，次键 Artist，第三键 Year。
	for _, col := range []string{"Year", "Artist", "Title"} {
		table.Click(col)
		fmt.Printf("click %-6s => keys %v\n", col, table.keys)
	}
	printTracks("MultiSorter（单趟多键排序）", table.tracks)

	// 用重复 sort.Stable 的方式排同样的键，结果应当一致。
	other := newTracks()
	sortByStable(other, table.keys)
	printTracks("sort.Stable（重复排序）", other)

	fmt.Printf("两种方式结果一致: %t\n", sameOrder(table.tracks, other))

	// 再次点击 Title：主键方向反转为降序。
	table.Click("Title")
	fmt.Printf("\nclick %-6s => keys %v\n", "Title", table.keys)
	printTracks("MultiSorter（Title 降序）", table.tracks)
}

func newTracks() []*Track {
	return []*Track{
		{"Go", "Delilah", "From the Roots Up", 2012, length("3m38s")},
		{"Go", "Moby", "Moby", 1992, length("3m37s")},
		{"Go", "Moby", "Moby Remixed", 1993, length("3m41s")},
		{"Go Ahead", "Alicia Keys", "As I Am", 2007, length("4m36s")},
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

func sameOrder(a, b []*Track) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if *a[i] != *b[i] {
			return false
		}
	}
	return true
}

func printTracks(title string, tracks []*Track) {
	fmt.Printf("\n== %s ==\n", title)
	const format = "%v\t%v\t%v\t%v\t%v\t\n"
	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(tw, format, "Title", "Artist", "Album", "Year", "Length")
	fmt.Fprintf(tw, format, "-----", "------", "-----", "----", "------")
	for _, t := range tracks {
		fmt.Fprintf(tw, format, t.Title, t.Artist, t.Album, t.Year, t.Length)
	}
	tw.Flush()
}
