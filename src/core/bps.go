package core

import "time"

// Maximum number of points used to calculate the average
var windowSize = 10

type progressPoint struct {
	time              time.Time
	totalBytesWritten uint64
}

type bytesPerSecond struct {
	timeStart time.Time
	points    []progressPoint
	counter   uint64 // Used to track bytes that are added in between seconds
}

func newBytesPerSecond() *bytesPerSecond {
	return &bytesPerSecond{timeStart: time.Now(), points: make([]progressPoint, 0)}
}

func (b *bytesPerSecond) addPoint(totalBytesWritten uint64) {
	var addPoint bool
	if len(b.points) == 0 {
		addPoint = true
	} else if (time.Since(b.lastPoint().time).Seconds()) > 1 {
		addPoint = true
	}
	if addPoint {
		b.counter += totalBytesWritten
		b.points = append(b.points, progressPoint{time: time.Now(), totalBytesWritten: b.counter})
		b.counter = 0
	} else {
		b.counter += totalBytesWritten
	}
}

func (b *bytesPerSecond) lastPoint() progressPoint {
	return (b.points)[len(b.points)-1]
}

func (b *bytesPerSecond) calc() uint64 {
	var tBytes uint64
	if len(b.points) == 0 {
		return 0
	}
	points := b.points
	end := len(b.points)
	if end > windowSize {
		points = b.points[end-windowSize : end]
	}
	for _, y := range points {
		tBytes += y.totalBytesWritten
	}
	return uint64(float64(tBytes / uint64(len(points))))
}

func (b *bytesPerSecond) calcFull() uint64 {
	var tBytes uint64
	if len(b.points) == 0 {
		return 0
	}
	for _, y := range b.points {
		tBytes += y.totalBytesWritten
	}
	return uint64(float64(tBytes/uint64(len(b.points))) / time.Since(b.timeStart).Seconds())
}
