//go:build XXXnobuild

package main

import "sync"

func main() {
	var ch chan struct{}
	var wg sync.WaitGroup

	select {
	case <-ch:
		println("ch")
	case <-wg:
		println("wg")
	}
}
