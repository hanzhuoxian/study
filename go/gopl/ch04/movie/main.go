package main

import (
	"encoding/json"
	"fmt"
)

type Movie struct {
	Title  string
	Year   int  `json:"released"`
	Color  bool `json:"color,omitempty"`
	Actors []string
}

func main() {
	var movies = []Movie{
		{Title: "Casablanca", Year: 1942, Color: true,
			Actors: []string{"Humphrey Bogart", "Ingrid Bergman"}},
		{Title: "Citizen Kane", Year: 1941, Color: false,
			Actors: []string{"Orson Welles", "Joseph Cotten"}},
		{Title: "The Godfather", Year: 1972, Color: true,
			Actors: []string{"Marlon Brando", "Al Pacino"}},
	}

	data, err := json.Marshal(movies)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", data)

	dataIndent, err := json.MarshalIndent(movies, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", dataIndent)
}
