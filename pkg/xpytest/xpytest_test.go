package xpytest_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainer/xpytest/pkg/pytest"
	"github.com/chainer/xpytest/pkg/xpytest"
	xpytest_proto "github.com/chainer/xpytest/proto"
)

func TestXpytest(t *testing.T) {
	ctx := context.Background()

	lock := sync.WaitGroup{}
	total := int64(0)
	running := int64(0)
	lock.Add(1)
	base := &pytest.Pytest{
		Executor: func(
			ctx context.Context, args []string, d time.Duration, x []string,
		) (*xpytest_proto.TestResult, error) {
			defer atomic.AddInt64(&running, -1)
			defer atomic.AddInt64(&total, 1)
			atomic.AddInt64(&running, 1)
			lock.Wait()
			return &xpytest_proto.TestResult{
				Status: xpytest_proto.TestResult_SUCCESS,
			}, nil
		},
	}
	xpt := xpytest.NewXpytest(base)
	for i := 0; i < 100; i++ {
		xpt.Tests = append(xpt.GetTests(), &xpytest_proto.TestQuery{
			Deadline: 1.0,
		})
	}

	testGroup := sync.WaitGroup{}
	testGroup.Add(1)
	go func() {
		defer testGroup.Done()
		if err := xpt.Execute(ctx, 3, 4, nil); err != nil {
			t.Fatalf("failed to execute: %s", err)
		}
	}()

	for running < 12 {
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	if running != 12 {
		t.Fatalf("# of running jobs is unexpected: %d", running)
	}

	lock.Done()
	testGroup.Wait()

	if total != 100 {
		t.Fatalf("# of jobs is unexpected: %d", total)
	}
}

func TestXpytestWithResourceMultiplier(t *testing.T) {
	ctx := context.Background()

	lock := sync.WaitGroup{}
	running := int64(0)
	lock.Add(1)
	base := &pytest.Pytest{
		Executor: func(
			ctx context.Context, args []string, d time.Duration, x []string,
		) (*xpytest_proto.TestResult, error) {
			defer atomic.AddInt64(&running, -1)
			atomic.AddInt64(&running, 1)
			lock.Wait()
			return &xpytest_proto.TestResult{
				Status: xpytest_proto.TestResult_SUCCESS,
			}, nil
		},
	}
	xpt := xpytest.NewXpytest(base)
	for i := 0; i < 100; i++ {
		xpt.Tests = append(xpt.GetTests(), &xpytest_proto.TestQuery{
			Deadline: 1.0,
			Resource: 2.0,
		})
	}

	testGroup := sync.WaitGroup{}
	testGroup.Add(1)
	go func() {
		defer testGroup.Done()
		if err := xpt.Execute(ctx, 3, 4, nil); err != nil {
			t.Fatalf("failed to execute: %s", err)
		}
	}()

	for running < 6 {
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	if running != 6 {
		t.Fatalf("# of running jobs is unexpected: %d", running)
	}

	lock.Done()
	testGroup.Wait()
}
