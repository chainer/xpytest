package pytest_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chainer/xpytest/pkg/pytest"
	xpytest_proto "github.com/chainer/xpytest/proto"
)

type pytestExecutor struct {
	// Input parameters.
	Args     []string
	Deadline time.Duration
	Env      []string

	// Output parameters.
	TestResult *xpytest_proto.TestResult
	Error      error
}

func (p *pytestExecutor) Execute(
	ctx context.Context,
	args []string, deadline time.Duration, env []string,
) (*xpytest_proto.TestResult, error) {
	p.Args = args
	p.Deadline = deadline
	p.Env = env
	if p.TestResult == nil {
		p.TestResult = &xpytest_proto.TestResult{}
	}
	return p.TestResult, p.Error
}

func TestPytest(t *testing.T) {
	ctx := context.Background()
	p := pytest.NewPytest("python3")
	executor := &pytestExecutor{
		TestResult: &xpytest_proto.TestResult{
			Status: xpytest_proto.TestResult_SUCCESS,
			Stdout: "=== 123 passed in 4.56 seconds ===",
		},
	}
	p.Executor = executor.Execute
	p.Files = []string{"test_foo.py"}
	p.Deadline = time.Minute
	if r, err := p.Execute(ctx); err != nil {
		t.Fatalf("failed to execute: %s", err)
	} else if strings.Join(executor.Args, ",") !=
		"python3,-m,pytest,test_foo.py" {
		t.Fatalf("unexpected args: %s", executor.Args)
	} else if executor.Env != nil {
		t.Fatalf("unexpected envs: %s", executor.Env)
	} else if s := r.Summary(); s !=
		"[SUCCESS] test_foo.py (123 passed in 4.56 seconds)" {
		t.Fatalf("unexpected summary: %s", s)
	}
}

func TestPytestWithXdist(t *testing.T) {
	ctx := context.Background()
	p := pytest.NewPytest("python3")
	executor := &pytestExecutor{
		TestResult: &xpytest_proto.TestResult{
			Status: xpytest_proto.TestResult_SUCCESS,
			Stdout: "=== 123 passed in 4.56 seconds ===",
		},
	}
	p.Executor = executor.Execute
	p.Files = []string{"test_foo.py"}
	p.Deadline = time.Minute
	p.Xdist = 4
	if _, err := p.Execute(ctx); err != nil {
		t.Fatalf("failed to execute: %s", err)
	} else if strings.Join(executor.Args, ",") !=
		"python3,-m,pytest,-n,4,test_foo.py" {
		t.Fatalf("unexpected args: %s", executor.Args)
	}
}

func TestPytestWhenAllTestsAreDeselected(t *testing.T) {
	ctx := context.Background()
	p := pytest.NewPytest("python3")
	executor := &pytestExecutor{
		TestResult: &xpytest_proto.TestResult{
			Status: xpytest_proto.TestResult_FAILED,
			Stdout: "=== 123 deselected in 1.23 seconds ===",
		},
	}
	p.Executor = executor.Execute
	p.Files = []string{"test_foo.py"}
	p.Deadline = time.Minute
	if r, err := p.Execute(ctx); err != nil {
		t.Fatalf("failed to execute: %s", err)
	} else if s := r.Summary(); s !=
		"[SUCCESS] test_foo.py (123 deselected in 1.23 seconds)" {
		t.Fatalf("unexpected summary: %s", s)
	}
}

func TestPytestWithFlakyTest(t *testing.T) {
	ctx := context.Background()
	p := pytest.NewPytest("python3")
	trial := 0
	p.Executor = func(
		ctx context.Context,
		args []string, deadline time.Duration, env []string,
	) (*xpytest_proto.TestResult, error) {
		trial++
		if trial == 1 {
			return &xpytest_proto.TestResult{
				Status: xpytest_proto.TestResult_FAILED,
				Stdout: "=== 1 failed, 122 passed in 1.23 seconds ===",
			}, nil
		}
		return &xpytest_proto.TestResult{
			Status: xpytest_proto.TestResult_SUCCESS,
			Stdout: "=== 123 passed in 4.56 seconds ===",
		}, nil
	}
	p.Files = []string{"test_foo.py"}
	p.Deadline = time.Minute
	p.Retry = 2
	if r, err := p.Execute(ctx); err != nil {
		t.Fatalf("failed to execute: %s", err)
	} else if s := r.Summary(); s !=
		"[FLAKY] test_foo.py"+
			" (1 failed, 122 passed in 1.23 seconds * 2 trials)" {
		t.Fatalf("unexpected summary: %s", s)
	}
}

func TestPytestWithTimeoutTest(t *testing.T) {
	ctx := context.Background()
	p := pytest.NewPytest("python3")
	executor := &pytestExecutor{
		TestResult: &xpytest_proto.TestResult{
			Status: xpytest_proto.TestResult_TIMEOUT,
			Time:   61.234,
		},
	}
	p.Executor = executor.Execute
	p.Files = []string{"test_foo.py"}
	p.Deadline = time.Minute
	if r, err := p.Execute(ctx); err != nil {
		t.Fatalf("failed to execute: %s", err)
	} else if s := r.Summary(); s !=
		"[TIMEOUT] test_foo.py (61 seconds)" {
		t.Fatalf("unexpected summary: %s", s)
	}
}

func TestPytestWithOutput(t *testing.T) {
	ctx := context.Background()
	p := pytest.NewPytest("python3")
	executor := &pytestExecutor{
		TestResult: &xpytest_proto.TestResult{
			Status: xpytest_proto.TestResult_FAILED,
			Stdout: "foo\nbar\nbaz\n=== 1 failed, 23 passed in 4.5 seconds ===",
			Stderr: "stderr",
		},
	}
	p.Executor = executor.Execute
	p.Files = []string{"test_foo.py"}
	p.Deadline = time.Minute
	if r, err := p.Execute(ctx); err != nil {
		t.Fatalf("failed to execute: %s", err)
	} else if s := r.Summary(); s !=
		"[FAILED] test_foo.py (1 failed, 23 passed in 4.5 seconds)" {
		t.Fatalf("unexpected summary: %s", s)
	} else if s := r.Output(); s !=
		"[FAILED] test_foo.py (1 failed, 23 passed in 4.5 seconds)\n"+
			"foo\nbar\nbaz\n"+
			"=== 1 failed, 23 passed in 4.5 seconds ===\n"+
			"stderr" {
		t.Fatalf("unexpected output: %s", s)
	}
}

func TestPytestWithLongOutput(t *testing.T) {
	ctx := context.Background()
	p := pytest.NewPytest("python3")
	executor := &pytestExecutor{
		TestResult: &xpytest_proto.TestResult{
			Status: xpytest_proto.TestResult_FAILED,
			Stdout: strings.Repeat("foo\n", 1000) +
				"=== 1 failed, 23 passed in 4.5 seconds ===",
		},
	}
	p.Executor = executor.Execute
	p.Files = []string{"test_foo.py"}
	p.Deadline = time.Minute
	if r, err := p.Execute(ctx); err != nil {
		t.Fatalf("failed to execute: %s", err)
	} else if s := r.Summary(); s !=
		"[FAILED] test_foo.py (1 failed, 23 passed in 4.5 seconds)" {
		t.Fatalf("unexpected summary: %s", s)
	} else if ss := strings.Split(r.Output(), "\n"); len(ss) != 502 &&
		ss[251] != "...(701 lines skipped)..." {
		t.Fatalf("unexpected output: %d: %s", len(ss), ss)
	}
}
