package resourcebuckets_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainer/xpytest/pkg/resourcebuckets"
)

func TestResourceBuckets(t *testing.T) {
	rb := resourcebuckets.NewResourceBuckets(2, 3)
	buckets := make([]int32, 2)
	wg := sync.WaitGroup{}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ru := rb.Acquire(1)
			v := atomic.AddInt32(&buckets[ru.Index], 1)
			if v > 3 {
				t.Errorf("exceeding capacity: %d", v)
			}
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&buckets[ru.Index], -1)
			rb.Release(ru)
		}()
	}
	wg.Wait()
}

func TestResourceBucketsIndexes(t *testing.T) {
	rb := resourcebuckets.NewResourceBuckets(4, 10)
	type TestCase struct {
		Capacity int
		Index    int
	}
	tcs := []TestCase{
		TestCase{Capacity: 1, Index: 0},
		TestCase{Capacity: 2, Index: 1},
		TestCase{Capacity: 3, Index: 2},
		TestCase{Capacity: 4, Index: 3},
		TestCase{Capacity: 5, Index: 0},
		TestCase{Capacity: 6, Index: 1},
		TestCase{Capacity: 7, Index: 2},
		TestCase{Capacity: 2, Index: 3},
		TestCase{Capacity: 2, Index: 0},
		TestCase{Capacity: 2, Index: 3},
		TestCase{Capacity: 1, Index: 0},
		TestCase{Capacity: 1, Index: 1},
		TestCase{Capacity: 1, Index: 3},
		TestCase{Capacity: 1, Index: 0},
		TestCase{Capacity: 1, Index: 1},
		TestCase{Capacity: 1, Index: 3},
	}
	for i, tc := range tcs {
		if idx := rb.Acquire(tc.Capacity).Index; idx != tc.Index {
			t.Errorf("[case #%d] unexpected index: actual=%d, expected=%d",
				i, idx, tc.Index)
		}
	}
}
