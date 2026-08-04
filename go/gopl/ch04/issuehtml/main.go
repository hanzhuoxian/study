package main

import (
	"html/template"
	"log"
	"os"
	"time"

	"github.com/hanzhuoxian/study/go/gopl/ch04/github"
)

const tpl = `<h1>{{.TotalCount}} issues</h1>
<table>
<tr style='text-align: left'>
  <th>#</th>
  <th>State</th>
  <th>User</th>
  <th>Title</th>
</tr>
{{range .Items}}
<tr>
  <td><a href='{{.HTMLURL}}'>{{.Number}}</a></td>
  <td>{{.State}}</td>
  <td><a href='{{.User.HTMLURL}}'>{{.User.Login}}</a></td>
  <td><a href='{{.HTMLURL}}'>{{.Title}}</a></td>
</tr>
{{end}}
</table>`

var reportTemplate = template.Must(template.New("report").Funcs(template.FuncMap{"daysAgo": daysAgo}).Parse(tpl))

func daysAgo(t time.Time) int {
	return int(time.Since(t).Hours() / 24)
}
func main() {
	results, err := github.SearchIssues([]string{"mall"})
	if err != nil {
		log.Fatal(err)
	}
	if results.TotalCount == 0 {
		log.Println("No issues found.")
		return
	}

	err = reportTemplate.Execute(os.Stdout, results)
	if err != nil {
		log.Fatal(err)
	}

}
