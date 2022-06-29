package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"go.lepak.sg/playground/tree/binary"
)

func main() {
	fmt.Print("in-order: ")
	in := readInts()
	fmt.Println(in)

	fmt.Print("pre-order: ")
	pre := readInts()
	fmt.Println(pre)

	fmt.Print("mode (i/r): ")
	var mode rune
	_, err := fmt.Scanf("%c\n", &mode)
	if err != nil {
		panic(err)
	}

	var impl func([]int, []int) (*binary.Tree[int], error)
	switch mode {
	case 'i':
		// interesting...
		// actual type params of the function cannot be inferred
		// even though the variable has the fully instantiated type
		impl = binary.BuildFromPreAndInOrderIter[[]int, int]
	case 'r':
		impl = binary.BuildFromPreAndInOrderRec[[]int, int]
	default:
		panic("not a valid mode")
	}

	tr, err := impl(pre, in)
	if err != nil {
		panic(err)
	}

	fmt.Println("tree:")
	fmt.Print(tr.String())
}

func readInts() []int {
	raw, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		panic(err)
	}

	raws := strings.Split(strings.TrimSpace(raw), " ")

	out := make([]int, len(raws))

	for i, rawNum := range raws {
		num, err := strconv.Atoi(rawNum)
		if err != nil {
			panic(err)
		}

		out[i] = num
	}
	return out
}
