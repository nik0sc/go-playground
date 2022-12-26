package main

import "go.lepak.sg/playground/slidingwindow"

func main() {
	c := slidingwindow.NewCounter[int](10, 0, nil)
	c.Observe(1)
	println(c.Get(1))
}
