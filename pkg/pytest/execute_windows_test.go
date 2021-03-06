package pytest_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/chainer/xpytest/pkg/pytest"
	xpytest_proto "github.com/chainer/xpytest/proto"
)

func TestExecute(t *testing.T) {
	ctx := context.Background()
	equivalentTrueCmd := []string{"cmd", "/c", "sort < NUL > NUL"}
	r, err := pytest.Execute(ctx, equivalentTrueCmd, time.Minute, nil)
	if err != nil {
		t.Fatalf("failed to execute: %s", err)
	}
	if r.Status != xpytest_proto.TestResult_SUCCESS {
		t.Fatalf("unexpected status: %s", r.Status)
	}
}

func TestExecuteWithFailure(t *testing.T) {
	ctx := context.Background()
	r, err := pytest.Execute(
		ctx, []string{"cmd", "/c", "powershell -Command exit 1"},
		time.Minute, nil)
	if err != nil {
		t.Fatalf("failed to execute: %s", err)
	}
	if r.Status != xpytest_proto.TestResult_FAILED {
		t.Fatalf("unexpected status: %s", r.Status)
	}
}

func TestExecuteWithNoTests(t *testing.T) {
	ctx := context.Background()
	r, err := pytest.Execute(
		ctx, []string{"cmd", "/c", "powershell -Command exit 5"},
		time.Minute, nil)
	if err != nil {
		t.Fatalf("failed to execute: %s", err)
	}
	if r.Status != xpytest_proto.TestResult_SUCCESS {
		t.Fatalf("unexpected status: %s", r.Status)
	}
}

func TestExecuteWithTimeout(t *testing.T) {
	ctx := context.Background()
	r, err := pytest.Execute(
		ctx, []string{"cmd", "/c", "ping localhost -n 10 > NUL"},
		time.Millisecond*100, nil)
	if err != nil {
		t.Fatalf("failed to execute: %s", err)
	}
	if r.Status != xpytest_proto.TestResult_TIMEOUT {
		t.Fatalf("unexpected status: %s", r.Status)
	}
}

func TestExecuteWithEnvironmentVariables(t *testing.T) {
	ctx := context.Background()
	r, err := pytest.Execute(
		ctx, []string{"cmd", "/c", "echo %HOGE%"},
		time.Minute, []string{"HOGE=PIYO"})
	if err != nil {
		t.Fatalf("failed to execute: %s", err)
	}
	if r.Status != xpytest_proto.TestResult_SUCCESS {
		t.Fatalf("unexpected status: %s", r.Status)
	}
	if r.Stdout != "PIYO\r\n" {
		t.Fatalf("unexpected output: %s", fmt.Sprintln("PIYO"))
	}
}
