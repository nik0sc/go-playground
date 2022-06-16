// Package counter counts occurrences of elements in slices
// and also returns their k most- or least-frequent elements.
// It also provides utility functions for working with counters.
package counter

// Counter counts occurrences of each element of the slice
// and returns a map of elements to their counts.
func Counter[S ~[]E, E comparable](slice S) map[E]int {
	c := make(map[E]int)

	for _, v := range slice {
		c[v]++
	}

	return c
}

// fold makes a copy of a, then folds b into it using the function f.
func fold[E comparable](a, b map[E]int, f func(a, b int) int) map[E]int {
	sum := make(map[E]int, len(a))

	for el, cnt := range a {
		sum[el] = cnt
	}

	for el, cnt := range b {
		sum[el] = f(sum[el], cnt)
	}

	return sum
}

// Add adds counter a and b together and returns a copy.
func Add[E comparable](a, b map[E]int) map[E]int {
	return fold(a, b, func(l, r int) int { return l + r })
}

// Subtract subtracts the counter b from a and returns a copy.
func Subtract[E comparable](a, b map[E]int) map[E]int {
	return fold(a, b, func(l, r int) int { return l - r })
}

// Total sums up all counts in the counter.
func Total[E comparable](ctr map[E]int) int {
	sum := 0

	for _, cnt := range ctr {
		sum += cnt
	}

	return sum
}
