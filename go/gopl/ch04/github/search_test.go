package github

import (
	"log"
	"testing"
)

func TestSearchIssues(t *testing.T) {
	result, err := SearchIssues([]string{"openai"})
	if err != nil {
		log.Fatal(err)
	}
	if result.TotalCount == 0 {
		t.Fail()
	}
}
