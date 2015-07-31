package main

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/ajstarks/svgo"
)

type svgStat struct {
	Score int
	Day   time.Time
}

func (s svgStat) DayString() string {
	return s.Day.Format("2006-01-02")
}

type svgMatrix [][]svgStat

func newSVGMatrix(stats []svgStat) svgMatrix {
	var dataBuffer svgMatrix
	var currentRow []svgStat
	for i, s := range stats {
		if i != 0 && i%7 == 0 {
			dataBuffer = append(dataBuffer, currentRow)
			currentRow = []svgStat{}
			// Fallthrough
		}
		currentRow = append(currentRow, s)
	}
	dataBuffer = append(dataBuffer, currentRow)
	// transpose stats here.
	d := make(svgMatrix, len(dataBuffer[0]))
	for _, statRow := range dataBuffer {
		for x, stat := range statRow {
			d[x] = append(d[x], stat)
		}
	}
	return d
}

func (m svgMatrix) NumColumns() int {
	if len(m) == 0 {
		return 0
	}
	return len(m[0])
}

func (m svgMatrix) NumRows() int {
	return len(m)
}

func (m svgMatrix) String() string {
	var str string
	for _, s := range m {
		str += fmt.Sprintln(s)
	}
	return str
}

func getSVGStatsTop(stats []svgStat) int {
	var max int
	for _, s := range stats {
		if s.Score > max {
			max = s.Score
		}
	}
	return max
}

// quartileBoundaries returns a sorted slice of quartile boundaries
// for stats. The first number in boundaries is *always* 0, even if
// stats does not contain 0, the next three numbers are the quartiles,
// and the last number is the maximum number in stat, disregarding
// outliers. The boundaries are inclusive (e.g. [0, 100, 200, 300,
// 400] means that the first boundary is between 0 and 100,
// inclusive.)
func quartileBoundaries(stats []svgStat) (boundaries []int) {
	switch len(stats) {
	case 0:
		return nil
	case 1:
		return []int{0, stats[0].Score, stats[0].Score, stats[0].Score, stats[0].Score}
	case 2:
		return []int{0, stats[0].Score, stats[1].Score, stats[1].Score, stats[1].Score}
	}
	median := func(ss []svgStat) (medianIndex, medianValue int) {
		index := len(ss) / 2
		return index, ss[index].Score
	}
	// 0 is always the first value in boundaries
	boundaries = append(boundaries, 0)
	// Get the second quartile (which is just the median).
	q2Index, q2 := median(stats)
	// Get the first quartile, which is the midpoint between the
	// lowest value in stats and q2. Exclude q2 from the
	// calculation.
	_, q1 := median(stats[:q2Index])
	// Append the first and second quartiles to boundaries.
	boundaries = append(boundaries, q1, q2)
	// Get the third quartile, which is the midpoint between the
	// highest value in stats and q2. Exclude q2 from the
	// calculation.
	_, q3 := median(stats[q2Index+1:])
	// Add the high and maximum value in stats.
	boundaries = append(boundaries, q3, getSVGStatsTop(stats))
	return boundaries
}

func quartile(stats []svgStat, score int) int {
	if score < 0 {
		return 0
	}
	var count int
	for _, q := range quartileBoundaries(stats) {
		if score > q {
			count++
		}
	}
	return count
}

var svgColors = [5]string{"#eeeeee", "#d6e685", "#8cc665", "#44a340", "#1e6823"}

func getSVGColor(stats []svgStat, score int) string {
	return svgColors[quartile(stats, score)]
}

func newSVGStats(start time.Time, dcgs []DayCommitGroup) []svgStat {
	start = BeginningOfDay(start)
	end := BeginningOfDay(time.Now().Add(24 * time.Hour))
	if start.After(end) {
		log.Panicf("start (%s) must be before the current time (%s)", start, end)
	}
	// Restrict start to 40 days before end.
	if end.Sub(start)/(24*time.Hour) > (40 * 24 * time.Hour) {
		start = BeginningOfDay(end.Add(-1 * 40 * 24 * time.Hour))
	}
	stats := make([]svgStat, end.Sub(start)/(24*time.Hour))
	day := start
	// HACK: We need to figure out how to pass timezones around!
	var loc *time.Location
	if len(dcgs) != 0 {
		loc = dcgs[0].Day.Location()
	} else {
		loc = time.UTC
	}
	for i := range stats {
		stats[i].Day = day
		day = time.Date(day.Year(), day.Month(), day.Day()+1, 0, 0, 0, 0, loc)
	}
	// Two loops, oh well @_@.
	var dcgIndex int
	for i := len(stats) - 1; i >= 0; i-- {
		if dcgIndex < len(dcgs) && stats[i].Day.Equal(dcgs[dcgIndex].Day) {
			stats[i].Score = len(dcgs[dcgIndex].Commits)
			dcgIndex++
		}
	}
	return stats
}

func CreateStreakSVG(u User, g Group, w io.Writer) error {
	commits, err := GetUserCommits(u, time.Time{})
	if err != nil {
		return err
	}
	loc, err := GetGroupLocation(g)
	if err != nil {
		return err
	}
	stats := newSVGStats(g.CreatedOn, DayCommitGroups(commits, loc))
	m := newSVGMatrix(stats)
	canvas := svg.New(w)
	width := 13*m.NumColumns() + 13
	height := 13*m.NumRows() + 13
	canvas.Start(width, height)
	var index int
	for y, row := range m {
		for x, point := range row {
			canvas.Rect((x*13)+14, (y*13)+14, 11, 11,
				fmt.Sprintf(`style="fill:%s"`, getSVGColor(stats, point.Score)),
				fmt.Sprintf(`data-count="%d"`, point.Score),
				fmt.Sprintf(`data-date="%s"`, point.DayString()))
			index += 10
		}
	}
	canvas.End()
	return nil
}
