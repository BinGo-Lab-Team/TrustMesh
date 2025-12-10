package main

import (
	"Analyzer/internal/tools"
	"fmt"
)

func main() {
	var folder string
	fmt.Printf("Enter volumes folder: ")

	_, err := fmt.Scanln(&folder)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	list, err := tools.FindBlockDirs(folder)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	for {
		var round int
		fmt.Printf("Enter round number: ")

		_, err := fmt.Scanln(&round)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}

		data, err := tools.CountProposalsForRound(list, round)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}

		for k, v := range data {
			fmt.Printf("%s: %d\n", k, v)
		}
	}
}
