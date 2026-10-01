package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	var input struct {
		Criteria []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
			Observation struct {
				Result         string `json:"result"`
				Source         string `json:"source"`
				TestedRevision string `json:"testedRevision"`
			} `json:"observation"`
		} `json:"criteria"`
	}
	json.NewDecoder(os.Stdin).Decode(&input)
	for _, criterion := range input.Criteria {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\n", criterion.ID, criterion.Description,
			criterion.Observation.Result, criterion.Observation.Source, criterion.Observation.TestedRevision)
	}
}
