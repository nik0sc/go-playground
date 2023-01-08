package slices

import (
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

func GroupBy[K comparable, E any, S ~[]E](
	data S, f func(E) K,
) [][]E {
	return groupby(data, f, false)
}

func GroupByStable[K comparable, E any, S ~[]E](
	data S, f func(E) K,
) [][]E {
	return groupby(data, f, true)
}

func GroupAndOrderBy[
	K constraints.Ordered, E any, S ~[]E,
](
	data S, f func(E) K,
) [][]E {
	out := GroupBy(data, f)

	slices.SortFunc(out, func(a, b []E) bool {
		return f(a[0]) < f(b[0])
	})

	return out
}

func groupby[
	K comparable, E any, S ~[]E,
](
	data S, f func(E) K, stable bool,
) (
	out [][]E,
) {
	groups := make(map[K]*[]E)

	for _, el := range data {
		key := f(el)
		group, ok := groups[key]
		if !ok {
			if stable {
				out = append(out, []E{})
				group = &out[len(out)-1]
			} else {
				group = new([]E)
			}

			groups[key] = group
		}
		*group = append(*group, el)
	}

	if !stable {
		out = make([][]E, len(groups))
		i := 0
		for _, group := range groups {
			out[i] = *group
			i++
		}
	}

	return
}
