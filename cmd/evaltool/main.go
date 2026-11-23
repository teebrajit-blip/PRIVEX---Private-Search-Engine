package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	privexeval "privex/internal/eval"
)

type JudgedQuery struct {
	Query    string         `json:"query"`
	Rankings map[string]int `json:"rankings"`
}

func main() {
	path := flag.String("judgments", "testdata/judgments.json", "Judged query JSON")
	flag.Parse()
	data, err := os.ReadFile(*path)
	if err != nil {
		fmt.Printf("evaluation judgments unavailable: %v\n", err)
		return
	}
	var judgments []JudgedQuery
	if err := json.Unmarshal(data, &judgments); err != nil {
		panic(err)
	}
	var total float64
	for _, judgment := range judgments {
		ranking := make([]string, 0, len(judgment.Rankings))
		for id := range judgment.Rankings {
			ranking = append(ranking, id)
		}
		total += privexeval.NDCGAt10(ranking, judgment.Rankings)
	}
	if len(judgments) > 0 {
		fmt.Printf("nDCG@10: %.6f\n", total/float64(len(judgments)))
	}
}
