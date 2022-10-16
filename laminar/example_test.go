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
		// Context cancellation should be respected when blocking
		// in a long-running operation. Here it's time.After, but
		// in practice this would be some kind of I/O.
		// In other words, pass ctx into http.NewRequestWithContext,
		// sql.DB.QueryContext, etc...
		select {
		case <-time.After(10 * time.Millisecond):
			one = 1
		case <-ctx.Done():
			fmt.Println("getOne context:" + ctx.Err().Error())
		}

		fmt.Println("getOne exits")
		return nil
	})

	getTwo := g.NewTask("getTwo", func(ctx context.Context) error {
		select {
		case <-time.After(20 * time.Millisecond):
			two = 2
		case <-ctx.Done():
			fmt.Println("getTwo context:" + ctx.Err().Error())
		}

		fmt.Println("getTwo exits")
		return nil
	})

	g.NewTask("sum", func(ctx context.Context) error {
		// If context is canceled before this starts,
		// or if getOne or getTwo return an error,
		// this won't start
		fmt.Println("sum starts")

		// This is race-free as the writes to one and two
		// happen-before the reads here
		fmt.Println(one + two)
		return nil
	}).After(getOne, getTwo)

	// The dependency graph looks like:
	// getOne -.
	//         |--> sum
	// getTwo -`

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

	fmt.Println()
	fmt.Println(g)
	fmt.Println()

	// Output:
	// Group: started=false
	// getOne [created] -> sum [created]
	// getTwo [created] -> sum [created]
	// sum [created] ->
	//
	// getOne exits
	// getTwo exits
	// sum starts
	// 3
	//
	// Group: started=true
	// getOne [finished] -> sum [finished]
	// getTwo [finished] -> sum [finished]
	// sum [finished] ->
}
