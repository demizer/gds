package conui

import (
	"testing"
	"time"
)

var hashingDialogSortBarsTests = []struct {
	bars   []*HashingProgressBar
	expect []*HashingProgressBar
	rounds int // Number of times SortBars() should be called
}{
	{
		// Test finished bars, but not sorted
		bars: []*HashingProgressBar{
			{Label: "test1-1", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now()},
			{Label: "test1-3", SizeWritn: 500, SizeTotal: 1000},
			{Label: "test1-2", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now()},
			{Label: "test1-4", SizeWritn: 1, SizeTotal: 1000},
			{Label: "test1-5", SizeWritn: 0, SizeTotal: 1000},
		},
		expect: []*HashingProgressBar{
			{Label: "test1-1", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test1-2", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test1-3", SizeWritn: 500, SizeTotal: 1000},
			{Label: "test1-4", SizeWritn: 1, SizeTotal: 1000},
			{Label: "test1-5", SizeWritn: 0, SizeTotal: 1000},
		},
	},
	{
		// Test finished bars, but not sorted
		bars: []*HashingProgressBar{
			{Label: "test1.1-1", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now()},
			{Label: "test1.1-4", SizeWritn: 500, SizeTotal: 1000},
			{Label: "test1.1-5", SizeWritn: 1, SizeTotal: 1000},
			{Label: "test1.1-2", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now()},
			{Label: "test1.1-6", SizeWritn: 0, SizeTotal: 1000},
			{Label: "test1.1-3", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now()},
		},
		expect: []*HashingProgressBar{
			{Label: "test1.1-1", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test1.1-2", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test1.1-3", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test1.1-4", SizeWritn: 500, SizeTotal: 1000},
			{Label: "test1.1-5", SizeWritn: 1, SizeTotal: 1000},
			{Label: "test1.1-6", SizeWritn: 0, SizeTotal: 1000},
		},
		// rounds: 2,
	},
	{
		// Test finished bars, with one sorted at first position with other unsorted bars
		bars: []*HashingProgressBar{
			{Label: "test1.2-1", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test1.2-4", SizeWritn: 500, SizeTotal: 1000},
			{Label: "test1.2-5", SizeWritn: 1, SizeTotal: 1000},
			{Label: "test1.2-2", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now()},
			{Label: "test1.2-6", SizeWritn: 0, SizeTotal: 1000},
			{Label: "test1.2-3", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now()},
		},
		expect: []*HashingProgressBar{
			{Label: "test1.2-1", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test1.2-2", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test1.2-3", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test1.2-4", SizeWritn: 500, SizeTotal: 1000},
			{Label: "test1.2-5", SizeWritn: 1, SizeTotal: 1000},
			{Label: "test1.2-6", SizeWritn: 0, SizeTotal: 1000},
		},
		rounds: 2,
	},
	{
		// Test finished bars, with one sorted at first position with other unsorted bars
		bars: []*HashingProgressBar{
			{Label: "test1.3-1", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test1.3-2", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test1.3-4", SizeWritn: 500, SizeTotal: 1000},
			{Label: "test1.3-5", SizeWritn: 1, SizeTotal: 1000},
			{Label: "test1.3-6", SizeWritn: 0, SizeTotal: 1000},
			{Label: "test1.3-3", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now()},
		},
		expect: []*HashingProgressBar{
			{Label: "test1.3-1", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test1.3-2", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test1.3-3", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test1.3-4", SizeWritn: 500, SizeTotal: 1000},
			{Label: "test1.3-5", SizeWritn: 1, SizeTotal: 1000},
			{Label: "test1.3-6", SizeWritn: 0, SizeTotal: 1000},
		},
		rounds: 2,
	},
	{
		// Two complete unsorted bars should be moved to the top of the list
		bars: []*HashingProgressBar{
			{Label: "test2-3", SizeWritn: 500, SizeTotal: 1000},
			{Label: "test2-1", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now()},
			{Label: "test2-2", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now()},
			{Label: "test2-4", SizeWritn: 0, SizeTotal: 1000},
			{Label: "test2-5", SizeWritn: 1, SizeTotal: 1000},
		},
		expect: []*HashingProgressBar{
			{Label: "test2-1", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test2-2", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test2-3", SizeWritn: 500, SizeTotal: 1000},
			{Label: "test2-4", SizeWritn: 0, SizeTotal: 1000},
			{Label: "test2-5", SizeWritn: 1, SizeTotal: 1000},
		},
	},
	{
		// No bars are finished, no sorting should be done
		bars: []*HashingProgressBar{
			{Label: "test3-1", SizeWritn: 900, SizeTotal: 1000},
			{Label: "test3-2", SizeWritn: 500, SizeTotal: 1000},
			{Label: "test3-3", SizeWritn: 100, SizeTotal: 1000},
			{Label: "test3-4", SizeWritn: 1, SizeTotal: 1000},
			{Label: "test3-5", SizeWritn: 0, SizeTotal: 1000},
		},
		expect: []*HashingProgressBar{
			{Label: "test3-1", SizeWritn: 900, SizeTotal: 1000},
			{Label: "test3-2", SizeWritn: 500, SizeTotal: 1000},
			{Label: "test3-3", SizeWritn: 100, SizeTotal: 1000},
			{Label: "test3-4", SizeWritn: 1, SizeTotal: 1000},
			{Label: "test3-5", SizeWritn: 0, SizeTotal: 1000},
		},
	},
	{
		// A finished bar immediately after sorted bars
		bars: []*HashingProgressBar{
			{Label: "test4-1", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test4-2", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test4-3", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now()},
			{Label: "test4-4", SizeWritn: 0, SizeTotal: 1000},
			{Label: "test4-5", SizeWritn: 1, SizeTotal: 1000},
		},
		expect: []*HashingProgressBar{
			{Label: "test4-1", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test4-2", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test4-3", SizeWritn: 1000, SizeTotal: 1000, timeFinished: time.Now(), sorted: true},
			{Label: "test4-4", SizeWritn: 0, SizeTotal: 1000},
			{Label: "test4-5", SizeWritn: 1, SizeTotal: 1000},
		},
	},
}

func TestSortBars(t *testing.T) {
	for _, y := range hashingDialogSortBarsTests {
		hd := &HashingDialog{Bars: y.bars}
		hde := y.expect
		for x := 0; x <= y.rounds; x++ {
			hd.SortBars()
		}
		for bx, bs := range hd.Bars {
			exp := hde[bx]
			sizeCheck := (bs.SizeWritn == exp.SizeWritn && bs.SizeTotal == exp.SizeTotal)
			timeCheck := (bs.timeFinished.Second() == exp.timeFinished.Second())
			sortedCheck := (bs.sorted == exp.sorted)
			if bs.Label != exp.Label || !sizeCheck || !timeCheck || !sortedCheck {
				t.Errorf("\nEXPECT:\t%s\nGOT:\t%s", exp, bs)
			}
			Log.Debugf("got: Label: %s SizeWritn: %d SizeTotal: %d timeFinished: %d sorted: %t",
				bs.Label, bs.SizeWritn, bs.SizeTotal, bs.timeFinished.Second(), bs.sorted)
			Log.Debugf("exp: Label: %s SizeWritn: %d SizeTotal: %d timeFinished: %d sorted: %t",
				exp.Label, exp.SizeWritn, exp.SizeTotal, exp.timeFinished.Second(), exp.sorted)
		}
	}
}
