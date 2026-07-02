package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var taskRmCmd = &cobra.Command{
	Use:        "rm <id>",
	Short:      "Remove a task",
	Deprecated: "use 'task done <id>' to complete a task instead.",
	Args:       cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Printf("invalid id %q: %v\n", args[0], err)
			return
		}
		removeTask(id)
	},
}

func removeTask(id int) {
	for i, t := range tasks {
		if t.ID != id {
			continue
		}
		tasks = append(tasks[:i], tasks[i+1:]...)
		fmt.Printf("task #%d [%s] removed\n", id, t.Name)
		return
	}
	fmt.Printf("task #%d not found\n", id)
}

func init() {
	taskCmd.AddCommand(taskRmCmd)
}
