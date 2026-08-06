package main

// **练习5.14：** 使用breadthFirst遍历其他数据结构。比如，topoSort例子中的课程依赖关系（有向图）、
// 个人计算机的文件层次结构（树）；你所在城市的公交或地铁线路（无向图）。

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	fmt.Println("=== 课程先修关系（有向图） ===")
	breadthFirst(courseNeighbors, []string{"compilers", "networks"})

	fmt.Println("\n=== 文件系统层次结构（树） ===")
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	breadthFirst(fileNeighbors, []string{dir})

	fmt.Println("\n=== 地铁线路（无向图） ===")
	breadthFirst(subwayNeighbors, []string{"人民广场"})
}

// breadthFirst 对 workList 中的元素调用 f 做广度优先遍历，f 返回的新元素会加入下一轮的待访问列表。
// seen 保证每个元素只被访问一次，因此对有向图、树和带环的无向图都能正确终止。
func breadthFirst(f func(item string) []string, workList []string) {
	seen := make(map[string]bool)
	for len(workList) > 0 {
		items := workList
		workList = nil
		for _, item := range items {
			if !seen[item] {
				seen[item] = true
				workList = append(workList, f(item)...)
			}
		}
	}
}

// preReqs 记录课程的先修课程（有向图），取自 topoSort 例子。
var preReqs = map[string][]string{
	"algorithms": {"data structures"},
	"calculus":   {"linear algebra"},
	"compilers": {
		"data structures",
		"formal languages",
		"computer organization",
	},
	"data structures":       {"discrete math"},
	"databases":             {"data structures"},
	"discrete math":         {"intro to programming"},
	"formal languages":      {"discrete math"},
	"networks":              {"operating systems"},
	"operating systems":     {"data structures", "computer organization"},
	"programming languages": {"data structures", "computer organization"},
}

// courseNeighbors 打印课程名并返回其先修课程列表。
func courseNeighbors(course string) []string {
	fmt.Println(course)
	return preReqs[course]
}

// fileNeighbors 打印目录名并返回其子目录列表（跳过隐藏目录），用于遍历文件系统树。
func fileNeighbors(dir string) []string {
	fmt.Println(dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Print(err)
		return nil
	}

	var subDirs []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name()[0] != '.' {
			subDirs = append(subDirs, filepath.Join(dir, entry.Name()))
		}
	}
	return subDirs
}

// subway 记录地铁站与其相邻站点的关系（无向图，边由两端各存一份实现）。
var subway = map[string][]string{
	"人民广场": {"陕西南路", "新天地"},
	"陕西南路": {"人民广场", "衡山路"},
	"新天地":  {"人民广场", "打浦桥"},
	"衡山路":  {"陕西南路"},
	"打浦桥":  {"新天地"},
}

// subwayNeighbors 打印地铁站名并返回其相邻站点列表。
func subwayNeighbors(station string) []string {
	fmt.Println(station)
	return subway[station]
}
