/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

type Task struct {
	ID        int
	Name      string
	Priority  string
	DueAt     time.Time
	Remind    bool
	Tags      []string
	CreatedAt time.Time
	Done      bool
}

var tasks []Task

// taskAddCmd represents the taskAdd command
var taskAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new task",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		priority, _ := cmd.Flags().GetString("priority")
		remind, _ := cmd.Flags().GetBool("remind")
		tags, _ := cmd.Flags().GetStringSlice("tags")
		due, _ := cmd.Flags().GetTime("due")
		task := addTask(args[0], priority, due, remind, tags)
		fmt.Printf("Task #%#v\n",
			task)
	},
}

var dueFormats = []string{
	"2006-01-02 15:04",
	"2006-01-02",
	"01-02 15:04",
}

func init() {
	rootCmd.AddCommand(taskAddCmd)

	taskAddCmd.Flags().StringP("priority", "p", "medium", "Task priority: low, medium, high")
	taskAddCmd.Flags().BoolP("remind", "r", false, "Enable reminder")
	taskAddCmd.Flags().TimeP("due", "d", time.Now().Add(1*time.Hour), dueFormats, "Due time, e.g. \"2006-01-02 15:04\"")
	taskAddCmd.Flags().StringSliceP("tags", "t", nil, "Comma-separated tags")

	taskAddCmd.MarkFlagRequired("priority")
	taskAddCmd.MarkFlagsRequiredTogether("remind", "due")

	taskAddCmd.RegisterFlagCompletionFunc("priority", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return []string{"low", "medium", "high"}, cobra.ShellCompDirectiveDefault
	})
}

func addTask(name, priority string, due time.Time, remind bool, tags []string) Task {
	task := Task{
		ID:        len(tasks) + 1,
		Name:      name,
		Priority:  priority,
		DueAt:     due,
		Remind:    remind,
		Tags:      tags,
		CreatedAt: time.Now(),
	}
	tasks = append(tasks, task)

	return task
}
