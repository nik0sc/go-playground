package main

import (
	"flag"
	"fmt"
	"time"

	"go.lepak.sg/playground/tree/binary"
)

var (
	seed     = flag.Int64("s", 0, "seed (default current unix time in ns)")
	num      = flag.Int("n", 10, "number of nodes in the tree")
	balanced = flag.Bool("b", false, "if true, keep building the tree until it is balanced")
)

func main() {
	flag.Parse()

	if *seed == 0 {
		*seed = time.Now().UnixNano()
	}

	var tr *binary.Tree[int]
	attempts := 0

	if *balanced {
		tr, attempts = binary.BuildRandomBalanced(*num, *seed)
	} else {
		tr = binary.BuildRandom(*num, *seed)
	}

	preorder := make([]int, 0, *num)
	tr.PreOrder(func(k int) bool {
		preorder = append(preorder, k)
		return true
	})

	inorder := make([]int, 0, *num)
	for n := range tr.InOrderCoroutine().Items() {
		inorder = append(inorder, n)
	}

	fmt.Println("preorder:", preorder)
	fmt.Println("inorder:", inorder)

	fmt.Println("tree:")
	fmt.Println(tr.String())

	actual, ideal := tr.Height()
	fmt.Println("height:", actual, "ideal:", ideal)

	if *balanced {
		fmt.Println("attempts:", attempts)
	}
}
