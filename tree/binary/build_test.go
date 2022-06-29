package binary

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var buildImpls = []struct {
	name  string
	fname string
	f     func(pre, in []int) (*Tree[int], error)
}{
	{
		name:  "iter",
		fname: "BuildFromPreAndInOrderIter",
		f:     BuildFromPreAndInOrderIter[[]int, int],
	},
	{
		name:  "rec",
		fname: "BuildFromPreAndInOrderRec",
		f:     BuildFromPreAndInOrderRec[[]int, int],
	},
}

func TestBuildFromPreAndInOrder(t *testing.T) {
	seedrd := rand.New(rand.NewSource(0x123456789abcdef0))
	const rounds = 100
	const size = 100

	for i := 0; i < rounds; i++ {
		seed := int64(seedrd.Uint64())
		tr := BuildRandom(size, seed)
		origStr := tr.String()
		//log.Print(origStr)
		var inOrder, preOrder []int

		tr.InOrder(func(k int) bool {
			inOrder = append(inOrder, k)
			return true
		})

		tr.PreOrder(func(k int) bool {
			preOrder = append(preOrder, k)
			return true
		})

		require.Equal(t, len(inOrder), len(preOrder), "different traversal length")

		origInOrder, origPreOrder := make([]int, len(inOrder)), make([]int, len(preOrder))
		copy(origInOrder, inOrder)
		copy(origPreOrder, preOrder)

		for _, impl := range buildImpls {
			t.Run(fmt.Sprintf("round=%d/%s", i, impl.name), func(t *testing.T) {
				trNew, err := impl.f(preOrder, inOrder)
				assert.NoError(t, err)
				assert.Equal(t, origStr, trNew.String(), "different tree was recreated")
				assert.Equal(t, origInOrder, inOrder, "inOrder was mutated")
				assert.Equal(t, origPreOrder, preOrder, "preOrder was mutated")
			})
		}
	}
}

var trForBench *Tree[int]

func BenchmarkBuildFromPreAndInOrder(b *testing.B) {
	seedrd := rand.New(rand.NewSource(0x123456789abcdef0))
	sizes := []int{10, 100, 10000, 1000000}

	for _, size := range sizes {
		tr := BuildRandom(size, int64(seedrd.Uint64()))
		var inOrder, preOrder []int

		tr.InOrder(func(k int) bool {
			inOrder = append(inOrder, k)
			return true
		})

		tr.PreOrder(func(k int) bool {
			preOrder = append(preOrder, k)
			return true
		})

		for _, impl := range buildImpls {
			b.Run(fmt.Sprintf("size=%d/%s", size, impl.name), func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					trForBench, _ = impl.f(preOrder, inOrder)
				}
			})
		}
	}
}
