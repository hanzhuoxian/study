package seeders

import "gohub/pkg/seed"

func Init() {
	seed.SetRunOrder([]string{
		"user",
	})
}
