package eval

import (
	"math"
	"sort"
)

type Judgment struct {
	Query   string
	Ratings map[string]int
}

func DCG(ratings []int) float64 {
	var total float64
	for i, rating := range ratings {
		if i == 0 {
			total += float64(rating)
			continue
		}
		total += float64(rating) / math.Log2(float64(i+2))
	}
	return total
}

func NDCGAt10(ranking []string, judgments map[string]int) float64 {
	got := make([]int, 10)
	for i, id := range ranking {
		if i >= 10 {
			break
		}
		got[i] = judgments[id]
	}
	ideal := make([]int, 0, len(judgments))
	for _, rating := range judgments {
		ideal = append(ideal, rating)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ideal)))
	if len(ideal) > 10 {
		ideal = ideal[:10]
	}
	denominator := DCG(ideal)
	if denominator == 0 {
		return 0
	}
	return DCG(got) / denominator
}

func ReciprocalRank(ranking []string, relevant map[string]bool) float64 {
	for i, id := range ranking {
		if relevant[id] {
			return 1.0 / float64(i+1)
		}
	}
	return 0
}
