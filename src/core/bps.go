package core

import "time"

type progressPoint struct {
	time              time.Time
	totalBytesWritten uint64
}

type bytesPerSecond struct {
	timeStart time.Time
	points    []progressPoint
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
		b.points = append(b.points, progressPoint{
			time:              time.Now(),
			totalBytesWritten: totalBytesWritten,
		})
	}
}

func (b *bytesPerSecond) lastPoint() progressPoint {
	return (b.points)[len(b.points)-1]
}

func (b *bytesPerSecond) calc() uint64 {
	var tBytes uint64
	var numPoints uint64 = 10
	if len(b.points) == 0 {
		return 0
	}
	for _, x := range b.points {
		tBytes += x.totalBytesWritten
	}
	numPoints = uint64(len(b.points))
	return uint64(float64(tBytes/numPoints) / time.Since(b.timeStart).Seconds())
}
