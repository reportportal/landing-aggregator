package buf

import (
	"container/ring"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
)

func TestRing(t *testing.T) {
	coffee := []string{"one", "two", "three"}

	// create a ring and populate it with some values
	r := ring.New(10)
	for i := 0; i < len(coffee); i++ {
		r.Value = coffee[i]
		r = r.Next()
	}

	// print all values non-nil of the ring, easy done with ring.Do()
	r.Do(func(x interface{}) {
		fmt.Println(x)
	})

	r.Value = "another"
	r = r.Next()

	// print all values of the ring, easy done with ring.Do()
	r.Do(func(x interface{}) {
		fmt.Println(x)
	})

	r.Value = "another2"
	r = r.Next()

	// print all values of the ring, easy done with ring.Do()
	r.Do(func(x interface{}) {
		fmt.Println(x)
	})

}

func BenchmarkRingBufferr_10000(b *testing.B) {
	buf := New(100)
	s1 := sync.WaitGroup{}
	s1.Add(b.N * 2)

	for n := 0; n < b.N; n++ {
		go func() {
			buf.Add(strconv.Itoa(rand.Int()))
			s1.Done()
		}()

		go func() {
			buf.Do(func(x interface{}) {
				fmt.Println(x)

			})
			s1.Done()
		}()
	}
	s1.Wait()
}
