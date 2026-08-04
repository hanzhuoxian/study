package main

import (
	"html/template"
	"log"
	"os"
	"time"

	"github.com/hanzhuoxian/study/go/gopl/ch04/github"
)

const tpl = `{{.TotalCount}} issues:
{{range .Items}}----------------------------------------
Number: {{.Number}}
User:   {{.User.Login}}
Title:  {{.Title | printf "%.64s"}}
Age:    {{.CreatedAt | daysAgo}} days
{{end}}`

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
