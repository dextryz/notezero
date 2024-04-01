package notezero

import (
	"log"
	"sort"
	"strings"
)

var (
	openBraket  = `<span class="highlight">`
	closeBraket = `</span>`
)

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func highlightIntervals(content string, highlights []string) [][]int {
	res := [][]int{}
	for _, v := range highlights {
		start := strings.Index(content, v)
		if start == -1 {
			// TODO pass event so we can properly log this.
			log.Println("highlight not found in article")
			continue
		}
		res = append(res, []int{start, start + len(v) - 1})
	}
	sort.Slice(res, func(i, j int) bool { return res[i][0] < res[j][0] })
	return res
}

func mergeIntervals(intervals [][]int) [][]int {

	if len(intervals) == 0 {
		return intervals
	}

	res := [][]int{intervals[0]}

	for _, v := range intervals[1:] {

		lastEnd := res[len(res)-1][1]

		start, end := v[0], v[1]

		if lastEnd < start {
			res = append(res, []int{start, end})
		} else {
			res[len(res)-1][1] = max(lastEnd, end)
		}
	}

	return res
}

func highlight(content string, intervals [][]int) string {

	if len(intervals) == 0 {
		return content
	}

	var res string
	lastIndex := 0
	for _, v := range intervals {
		start, end := v[0], v[1]+1
		res += content[lastIndex:start]
		res += openBraket + content[start:end] + closeBraket
		lastIndex = end
	}
	res += content[lastIndex:]
	return res
}
