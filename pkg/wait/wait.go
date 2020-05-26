package wait

import "time"

// Wait sec seconds
func Wait(sec int) {
	time.Sleep(time.Duration(sec) * time.Second)
}
