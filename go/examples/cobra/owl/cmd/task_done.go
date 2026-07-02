package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var taskDoneCmd = &cobra.Command{
	Use:   "done <id> [id...]",
	Short: "Mark one or more tasks as done",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")

		for _, arg := range args {
			id, err := strconv.Atoi(arg)
			if err != nil {
				fmt.Printf("invalid id %q: %v\n", arg, err)
				continue
			}
			markDone(id, force)
		}
	},
}

func markDone(id int, force bool) {
	for i := range tasks {
		if tasks[i].ID != id {
			continue
		}
		if tasks[i].Done && !force {
			fmt.Printf("task #%d already done (use --force to re-mark)\n", id)
			return
		}
		tasks[i].Done = true
		fmt.Printf("task #%d [%s] marked as done\n", id, tasks[i].Name)
		return
	}
	fmt.Printf("task #%d not found\n", id)
}

func init() {
	taskCmd.AddCommand(taskDoneCmd)

	// --force 是 done 的本地 flag，其他子命令看不到它
	taskDoneCmd.Flags().BoolP("force", "f", false, "Re-mark already-done tasks without warning")
}
