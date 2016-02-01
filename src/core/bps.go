package core

import "time"

// Maximum number of points used to calculate the average
var windowSize = 10

// ProgressPoint is a progress point containing bytes written in the last second added to the bps tracker.
type ProgressPoint struct {
	Time              time.Time // The time the point was added
	TotalBytesWritten uint64    // Total bytes written since the last update
}

// BytesPerSecond is used to calculate bytes per second transfer speeds using the average of the last ten points. A point
// should be added every second for accurate calculation.
type BytesPerSecond struct {
	TimeStart time.Time        // The time the bps tracker was initialized
	Points    []*ProgressPoint // Used to do the calculation
	counter   uint64           // Used to track bytes that are added in between seconds
}

// NewBytesPerSecond returns a new bytes per second object that can be used to track bytes per second transfer speeds.
func NewBytesPerSecond() *BytesPerSecond {
	return &BytesPerSecond{TimeStart: time.Now(), Points: make([]*ProgressPoint, 0)}
}

// AddPoint adds a new point to the progress points. It is initialized with using the totalBytesWritten argument. This should
// be called once a second for accurate results.
func (b *BytesPerSecond) AddPoint(totalBytesWritten uint64) {
	var addPoint bool
	if len(b.Points) == 0 {
		addPoint = true
	} else if (time.Since(b.LastPoint().Time).Seconds()) > 1 {
		addPoint = true
	}
	if addPoint {
		b.counter += totalBytesWritten
		b.Points = append(b.Points, &ProgressPoint{Time: time.Now(), TotalBytesWritten: b.counter})
		b.counter = 0
	} else {
		b.counter += totalBytesWritten
	}
}

// LastPoint returns the last progress point.
func (b *BytesPerSecond) LastPoint() *ProgressPoint {
	return (b.Points)[len(b.Points)-1]
}

// Calc returns the average bps calculation using the last 10 points.
func (b *BytesPerSecond) Calc() uint64 {
	var tBytes uint64
	if len(b.Points) == 0 {
		return 0
	}
	points := b.Points
	end := len(b.Points)
	if end > windowSize {
		points = b.Points[end-windowSize : end]
	}
	for _, y := range points {
		tBytes += y.TotalBytesWritten
	}
	return uint64(float64(tBytes / uint64(len(points))))
}

// CalcFull returns the average bps since time start including all points.
func (b *BytesPerSecond) CalcFull() uint64 {
	var tBytes uint64
	if len(b.Points) == 0 {
		return 0
	}
	for _, y := range b.Points {
		tBytes += y.TotalBytesWritten
	}
	return uint64(float64(tBytes / uint64(len(b.Points))))
}
