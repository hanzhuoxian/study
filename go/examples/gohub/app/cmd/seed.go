package cmd

import (
	"gohub/database/seeders"
	"gohub/pkg/console"
	"gohub/pkg/seed"

	"github.com/spf13/cobra"
)

var CmdDBSeed = &cobra.Command{
	Use:   "seed",
	Short: "Insert fake data to the database",
	Run:   runSeeders,
	Args:  cobra.MaximumNArgs(1),
}

func runSeeders(cmd *cobra.Command, args []string) {
	seeders.Init()
	if len(args) > 0 {
		name := args[0]
		seeder := seed.GetSeeder(name)
		if len(seeder.Name) > 0 {
			seed.RunSeeder(name)
		}
	} else {
		seed.RunAll()
		console.Success("Done seeding")
	}
}
