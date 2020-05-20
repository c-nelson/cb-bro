package main

import "time"

// StopWatch is used to tell elapsed time
type StopWatch struct {
	Start time.Time
}

// Elapsed returns the amount of time since start
func (sw *StopWatch) Elapsed() time.Duration {
	return time.Now().Sub(sw.Start)
}

// Reset start to now
func (sw *StopWatch) Reset() {
	sw.Start = time.Now()
}
