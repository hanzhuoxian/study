/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var taskListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "List tasks",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		all, _ := cmd.Flags().GetBool("all")
		priority, _ := cmd.Flags().GetString("priority")
		tag, _ := cmd.Flags().GetString("tag")

		addTask("读书", "medium", time.Now().Add(1*time.Hour), true, []string{"read"})

		filtered := tasks

		if !all {
			filtered = filterByPriority(filtered, priority)
		}
		if tag != "" {
			filtered = filterByTag(filtered, tag)
		}

		if len(filtered) == 0 {
			fmt.Println("No tasks found.")
			return
		}

		for _, t := range filtered {
			fmt.Printf("#%d  %-20s  priority=%-6s  due=%s  tags=[%s]\n",
				t.ID, t.Name, t.Priority,
				t.DueAt.Format("2006-01-02 15:04"),
				strings.Join(t.Tags, ","))
		}
	},
}

func filterByPriority(ts []Task, priority string) []Task {
	if priority == "" || priority == "all" {
		return ts
	}
	var out []Task
	for _, t := range ts {
		if strings.EqualFold(t.Priority, priority) {
			out = append(out, t)
		}
	}
	return out
}

func filterByTag(ts []Task, tag string) []Task {
	var out []Task
	for _, t := range ts {
		for _, tg := range t.Tags {
			if strings.EqualFold(tg, tag) {
				out = append(out, t)
				break
			}
		}
	}
	return out
}

func init() {
	taskCmd.AddCommand(taskListCmd)

	taskListCmd.Flags().BoolP("all", "a", false, "List all tasks regardless of priority")
	taskListCmd.Flags().StringP("priority", "p", "", "Filter by priority: low, medium, high")
	taskListCmd.Flags().StringP("tag", "g", "", "Filter by tag")

	// --all 和 --priority 互斥
	taskListCmd.MarkFlagsMutuallyExclusive("all", "priority")

	// --priority 补全
	taskListCmd.RegisterFlagCompletionFunc("priority", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return []string{"low", "medium", "high", "all"}, cobra.ShellCompDirectiveDefault
	})

	// --tag 动态补全：从内存中已有任务收集 tag
	taskListCmd.RegisterFlagCompletionFunc("tag", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		seen := map[string]struct{}{}
		var completions []string
		for _, t := range tasks {
			for _, tg := range t.Tags {
				if _, ok := seen[tg]; !ok {
					seen[tg] = struct{}{}
					completions = append(completions, tg)
				}
			}
		}
		return completions, cobra.ShellCompDirectiveDefault
	})
}
