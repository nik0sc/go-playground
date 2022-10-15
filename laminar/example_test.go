package laminar

import (
	"context"
	"fmt"
	"time"
)

func ExampleGroup() {
	// This example simulates long-running operations to get two values
	// then sums them up after both operations complete.
	g := NewGroup(context.Background(), NoLimit)

	var one, two int

	getOne := g.NewTask("getOne", func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		one = 1
		fmt.Println("getOne exits")
		return nil
	})

	getTwo := g.NewTask("getTwo", func(ctx context.Context) error {
		time.Sleep(20 * time.Millisecond)
		two = 2
		fmt.Println("getTwo exits")
		return nil
	})

	sum := g.NewTask("sum", func(ctx context.Context) error {
		fmt.Println("sum starts")
		fmt.Println(one + two)
		return nil
	})

	sum.After(getOne)
	sum.After(getTwo)

	// This will print the group state
	// as well as its task relationships
	fmt.Println(g)
	fmt.Println()

	err := g.Start()
	if err != nil {
		panic(err)
	}

	err = g.Wait()
	if err != nil {
		panic(err)
	}

	// Output:
	// Group: started=false
	// getOne -> sum
	// getTwo -> sum
	// sum ->
	//
	// getOne exits
	// getTwo exits
	// sum starts
	// 3
}
