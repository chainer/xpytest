package xpytest

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bmatcuk/doublestar"

	"github.com/chainer/xpytest/pkg/pytest"
	"github.com/chainer/xpytest/pkg/reporter"
	"github.com/chainer/xpytest/pkg/resourcebuckets"
	xpytest_proto "github.com/chainer/xpytest/proto"
)

// Xpytest is a controller for pytest queries.
type Xpytest struct {
	PytestBase  *pytest.Pytest
	Tests       []*xpytest_proto.TestQuery
	TestResults []*xpytest_proto.TestResult
	Status      xpytest_proto.TestResult_Status
}

// NewXpytest creates a new Xpytest.
func NewXpytest(base *pytest.Pytest) *Xpytest {
	return &Xpytest{PytestBase: base}
}

// GetTests returns test queries.
func (x *Xpytest) GetTests() []*xpytest_proto.TestQuery {
	if x.Tests == nil {
		return []*xpytest_proto.TestQuery{}
	}
	return x.Tests
}

// AddTestsWithFilePattern adds test files based on the given file pattern.
func (x *Xpytest) AddTestsWithFilePattern(pattern string) error {
	files, err := doublestar.Glob(pattern)
	if err != nil {
		return fmt.Errorf(
			"failed to find files with pattern: %s: %s", pattern, err)
	}
	for _, f := range files {
		if regexp.MustCompile(`^test_.*\.py$`).MatchString(path.Base(f)) {
			x.Tests = append(x.GetTests(), &xpytest_proto.TestQuery{File: f})
		}
	}
	return nil
}

// ApplyHint applies hint information to test cases.
// CAVEAT: This computation order is O(n^2).  This can be improved by sorting by
// suffixes.
func (x *Xpytest) ApplyHint(h *xpytest_proto.HintFile) error {
	for i := range h.GetSlowTests() {
		priority := i + 1
		hint := h.GetSlowTests()[len(h.GetSlowTests())-i-1]
		for _, tq := range x.GetTests() {
			if tq.GetFile() == hint.GetName() ||
				strings.HasSuffix(tq.GetFile(), "/"+hint.GetName()) {
				tq.Priority = int32(priority)
				if hint.GetDeadline() != 0 {
					tq.Deadline = hint.GetDeadline()
				} else {
					tq.Deadline = 600.0
				}
				if hint.GetXdist() != 0 {
					tq.Xdist = hint.GetXdist()
				}
				if hint.GetRetry() > 0 {
					tq.Retry = hint.GetRetry()
				}
			}
		}
	}
	return nil
}

// Execute runs tests.
func (x *Xpytest) Execute(
	ctx context.Context, bucket int, thread int,
	reporter reporter.Reporter,
) error {
	tests := append([]*xpytest_proto.TestQuery{}, x.Tests...)

	sort.SliceStable(tests, func(i, j int) bool {
		a, b := tests[i], tests[j]
		if a.Priority == b.Priority {
			return a.File < b.File
		}
		return a.Priority > b.Priority
	})

	if thread == 0 {
		thread = (runtime.NumCPU() + bucket - 1) / bucket
	}
	rb := resourcebuckets.NewResourceBuckets(bucket, thread)
	resultChan := make(chan *pytest.Result, thread)

	printer := sync.WaitGroup{}
	printer.Add(1)
	go func() {
		defer printer.Done()
		passedTests := []*pytest.Result{}
		flakyTests := []*pytest.Result{}
		failedTests := []*pytest.Result{}
		for {
			r, ok := <-resultChan
			if !ok {
				break
			}
			fmt.Println(r.Output())
			if r.Status == xpytest_proto.TestResult_SUCCESS {
				passedTests = append(passedTests, r)
			} else if r.Status == xpytest_proto.TestResult_FLAKY {
				flakyTests = append(flakyTests, r)
			} else {
				failedTests = append(failedTests, r)
			}
		}
		x.Status = xpytest_proto.TestResult_SUCCESS
		if len(flakyTests) > 0 {
			fmt.Printf("\n%s\n", horizon("FLAKY TESTS"))
			for _, t := range flakyTests {
				fmt.Printf("%s\n", t.Summary())
				if reporter != nil {
					reporter.Log(ctx, t.Summary())
				}
			}
			x.Status = xpytest_proto.TestResult_FLAKY
		}
		if len(failedTests) > 0 {
			fmt.Printf("\n%s\n", horizon("FAILED TESTS"))
			for _, t := range failedTests {
				fmt.Printf("%s\n", t.Summary())
			}
			x.Status = xpytest_proto.TestResult_FAILED
		}
		fmt.Printf("\n%s\n", horizon("TEST SUMMARY"))
		fmt.Printf("%d failed, %d flaky, %d passed\n",
			len(failedTests), len(flakyTests), len(passedTests))
	}()

	wg := sync.WaitGroup{}
	for _, t := range tests {
		t := t
		usage := rb.Acquire(func() int {
			if t.Xdist > 0 {
				return int(t.Xdist)
			}
			return 1
		}())
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer rb.Release(usage)
			pt := *x.PytestBase
			pt.Files = []string{t.File}
			pt.Xdist = int(t.Xdist)
			if pt.Xdist > thread {
				pt.Xdist = thread
			}
			if t.Retry != 0 {
				pt.Retry = int(t.Retry)
			}
			pt.Env = []string{
				fmt.Sprintf("CUDA_VISIBLE_DEVICES=%s", func() string {
					s := []string{}
					for i := 0; i < bucket; i++ {
						s = append(s, fmt.Sprintf("%d", (i+usage.Index)%bucket))
					}
					return strings.Join(s, ",")
				}()),
			}
			if t.Deadline != 0 {
				pt.Deadline = time.Duration(t.Deadline*1e6) * time.Microsecond
			}
			r, err := pt.Execute(ctx)
			if err != nil {
				panic(fmt.Sprintf("failed execute pytest: %s: %s", t.File, err))
			}
			resultChan <- r
		}()
	}
	wg.Wait()
	close(resultChan)
	printer.Wait()
	return nil
}

func horizon(title string) string {
	if title == "" {
		return strings.Repeat("=", 70)
	}
	title = " " + strings.TrimSpace(title) + " "
	s := strings.Repeat("=", (70-len(title))/2) + title
	return s + strings.Repeat("=", 70-len(s))
}
