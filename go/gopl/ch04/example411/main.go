package main

// 练习 4.11： 编写一个工具，允许用户在命令行创建、读取、更新和关闭GitHub上的issue，当必要的时候自动打开用户默认的编辑器用于输入文本信息。
//
// 用法：
//
//	issue create -repo owner/name -title "标题" [-body "正文"] [-labels a,b] [-assignees x,y]
//	issue read   -repo owner/name <编号>
//	issue update -repo owner/name <编号> [-title "标题"] [-body "正文"] [-state open|closed] [-labels a,b] [-assignees x,y] [-edit]
//	issue close  -repo owner/name <编号>
//
// owner/name 由用户在命令行显式给出（-repo），不是通过搜索获得的。
// create 未提供 -body、或 update 传入 -edit 时，会打开 $EDITOR（默认 vi）让用户输入正文。

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hanzhuoxian/study/go/gopl/ch04/github"
)

var (
	repo      = flag.String("repo", "", "目标仓库，格式 owner/name（必填）")
	title     = flag.String("title", "", "issue 标题（create 必填）")
	body      = flag.String("body", "", "issue 正文（留空则打开编辑器输入）")
	state     = flag.String("state", "", "新状态：open 或 closed（update 用）")
	labels    = flag.String("labels", "", "逗号分隔的标签列表")
	assignees = flag.String("assignees", "", "逗号分隔的指派人列表")
	edit      = flag.Bool("edit", false, "打开 $EDITOR 编辑正文（update 用，会用当前 body 预填充）")
)

func main() {
	if len(os.Args) < 2 {
		fatalf("用法: issue <create|read|update|close> -repo owner/name [flags] [编号]")
	}
	cmd := os.Args[1]
	flag.CommandLine.Parse(os.Args[2:]) // 跳过子命令词，解析其余 flag

	owner, name := splitRepo(*repo)

	switch cmd {
	case "create":
		runCreate(owner, name)
	case "read":
		runRead(owner, name)
	case "update":
		runUpdate(owner, name)
	case "close":
		runClose(owner, name)
	default:
		fatalf("unknown command: %s", cmd)
	}
}

func runCreate(owner, name string) {
	if *title == "" {
		fatalf("create: -title 是必填项")
	}

	issueBody := *body
	if issueBody == "" {
		var err error
		issueBody, err = readFromEditor("")
		if err != nil {
			fatalf("打开编辑器失败: %v", err)
		}
	}

	issue, err := Create(owner, name, &github.IssueRequest{
		Title:     *title,
		Body:      issueBody,
		Labels:    splitCSV(*labels),
		Assignees: splitCSV(*assignees),
	})
	if err != nil {
		fatalf("创建 issue 失败: %v", err)
	}
	printIssue(issue)
}

func runRead(owner, name string) {
	number := parseNumber()
	issue, err := github.GetIssue(owner, name, number)
	if err != nil {
		fatalf("读取 issue 失败: %v", err)
	}
	printIssue(issue)
}

func runUpdate(owner, name string) {
	number := parseNumber()

	if *state != "" && *state != "open" && *state != "closed" {
		fatalf("update: -state 必须是 open 或 closed，收到 %q", *state)
	}

	issueBody := *body
	if *edit {
		current, err := github.GetIssue(owner, name, number)
		if err != nil {
			fatalf("读取原始 issue 失败: %v", err)
		}
		issueBody, err = readFromEditor(current.Body)
		if err != nil {
			fatalf("打开编辑器失败: %v", err)
		}
	}

	issue, err := github.UpdateIssue(owner, name, number, &github.IssueRequest{
		Title:     *title,
		Body:      issueBody,
		State:     *state,
		Labels:    splitCSV(*labels),
		Assignees: splitCSV(*assignees),
	})
	if err != nil {
		fatalf("更新 issue 失败: %v", err)
	}
	printIssue(issue)
}

func runClose(owner, name string) {
	number := parseNumber()
	issue, err := github.CloseIssue(owner, name, number)
	if err != nil {
		fatalf("关闭 issue 失败: %v", err)
	}
	printIssue(issue)
}

// Create 是 github.CreateIssue 的薄封装。
func Create(owner, repo string, req *github.IssueRequest) (*github.Issue, error) {
	return github.CreateIssue(owner, repo, req)
}

// splitRepo 将 "owner/name" 形式的 -repo 参数拆分为 owner 和 name。
func splitRepo(repo string) (owner, name string) {
	if repo == "" {
		fatalf("-repo 是必填项，格式为 owner/name")
	}
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		fatalf("-repo 格式错误，应为 owner/name，收到 %q", repo)
	}
	return parts[0], parts[1]
}

// parseNumber 从剩余的非 flag 参数中取出唯一的 issue 编号。
func parseNumber() int {
	args := flag.Args()
	if len(args) != 1 {
		fatalf("需要且只能提供一个 issue 编号")
	}
	n, err := strconv.Atoi(args[0])
	if err != nil {
		fatalf("issue 编号必须是数字，收到 %q", args[0])
	}
	return n
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// readFromEditor 打开 $EDITOR（未设置时回退到 vi）编辑一个用 initial 预填充的
// 临时文件，返回用户保存后的内容。
func readFromEditor(initial string) (string, error) {
	tmp, err := os.CreateTemp("", "issue-*.md")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(initial); err != nil {
		tmp.Close()
		return "", fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("关闭临时文件失败: %w", err)
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, tmp.Name())
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("运行编辑器 %s 失败: %w", editor, err)
	}

	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		return "", fmt.Errorf("读取临时文件失败: %w", err)
	}
	return string(data), nil
}

func printIssue(issue *github.Issue) {
	fmt.Printf("#%d %s\n", issue.Number, issue.Title)
	fmt.Printf("状态: %s\n", issue.State)
	if issue.User != nil {
		fmt.Printf("作者: %s\n", issue.User.Login)
	}
	fmt.Printf("链接: %s\n", issue.HTMLURL)
	fmt.Printf("创建于: %s\n", issue.CreatedAt.Local().Format("2006-01-02 15:04:05"))
	if issue.Body != "" {
		fmt.Printf("\n%s\n", issue.Body)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
