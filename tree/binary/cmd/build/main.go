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

	tr := binary.BuildFromPreAndInOrder(pre, in)

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
