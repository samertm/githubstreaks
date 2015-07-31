package main

import "testing"

func TestQuartileBoundaries(t *testing.T) {
	makeStats := func(is ...int) []svgStat {
		var ss []svgStat
		for _, i := range is {
			ss = append(ss, svgStat{Score: i})
		}
		return ss
	}
	equalInts := func(is0, is1 []int) bool {
		if len(is0) != len(is1) {
			return false
		}
		for idx := range is0 {
			if is0[idx] != is1[idx] {
				return false
			}
		}
		return true
	}
	data := []struct {
		stats []svgStat
		want  []int
	}{{
		stats: makeStats(1, 2, 3, 4),
		want:  []int{0, 2, 3, 4, 4},
	}, {
		stats: makeStats(1, 2, 3),
		want:  []int{0, 1, 2, 3, 3},
	}}
	for _, d := range data {
		if got := quartileBoundaries(d.stats); !equalInts(d.want, got) {
			t.Errorf("Wanted %v, got %v", d.want, got)
		}
	}
}
