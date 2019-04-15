package resourcebuckets

import "sync"

// ResourceBuckets manages resource capacities.  This enables worker threads to
// use limited resources wihtout exceeding their capacities.
type ResourceBuckets struct {
	buckets    []int
	nextBucket int
	cond       *sync.Cond
}

// ResourceUsage represents a usage of resource.
type ResourceUsage struct {
	Index int
	Usage int
}

// NewResourceBuckets createa a new ResourceBuckets with buckets, each of which
// has the same size of capacity.
func NewResourceBuckets(size, capacityPerBucket int) *ResourceBuckets {
	buckets := make([]int, size)
	for i := range buckets {
		buckets[i] = capacityPerBucket
	}
	return &ResourceBuckets{
		buckets: buckets,
		cond:    sync.NewCond(&sync.Mutex{}),
	}
}

// Acquire acquires the given size of usage.  This function blocks until the
// size of usage can be acquired from the resources.
func (rb *ResourceBuckets) Acquire(usage int) *ResourceUsage {
	rb.cond.L.Lock()
	defer rb.cond.L.Unlock()
	for rb.buckets[rb.nextBucket] < usage {
		rb.cond.Wait()
	}
	ru := &ResourceUsage{Index: rb.nextBucket, Usage: usage}
	rb.buckets[ru.Index] -= ru.Usage
	rb.setNextBucket()
	return ru
}

// Release releases the given resource usage.
func (rb *ResourceBuckets) Release(ru *ResourceUsage) {
	rb.cond.L.Lock()
	defer rb.cond.L.Unlock()
	rb.buckets[ru.Index] += ru.Usage
	ru.Usage = 0
	rb.setNextBucket()
	rb.cond.Broadcast()
}

// setNextBucket recalculates rb.nextBucket.
// CAVEAT: rb.cond.L must be locked when this is called.
func (rb *ResourceBuckets) setNextBucket() {
	nextBucket := 0
	for i, b := range rb.buckets {
		if rb.buckets[nextBucket] < b {
			nextBucket = i
		}
	}
	rb.nextBucket = nextBucket
}
