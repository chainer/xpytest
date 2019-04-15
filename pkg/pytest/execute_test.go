package pytest_test

import (
	"context"
	"testing"
	"time"

	"github.com/chainer/xpytest/pkg/pytest"
	xpytest_proto "github.com/chainer/xpytest/proto"
)

func TestExecute(t *testing.T) {
	ctx := context.Background()
	r, err := pytest.Execute(ctx, []string{"true"}, time.Second, nil)
	if err != nil {
		t.Fatalf("failed to execute: %s", err)
	}
	if r.Status != xpytest_proto.TestResult_SUCCESS {
		t.Fatalf("unexpected status: %s", r.Status)
	}
}

func TestExecuteWithFailure(t *testing.T) {
	ctx := context.Background()
	r, err := pytest.Execute(ctx, []string{"false"}, time.Second, nil)
	if err != nil {
		t.Fatalf("failed to execute: %s", err)
	}
	if r.Status != xpytest_proto.TestResult_FAILED {
		t.Fatalf("unexpected status: %s", r.Status)
	}
}

func TestExecuteWithTimeout(t *testing.T) {
	ctx := context.Background()
	r, err := pytest.Execute(
		ctx, []string{"sleep", "10"}, 100*time.Millisecond, nil)
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
		ctx, []string{"bash", "-c", "echo $HOGE"},
		time.Second, []string{"HOGE=PIYO"})
	if err != nil {
		t.Fatalf("failed to execute: %s", err)
	}
	if r.Status != xpytest_proto.TestResult_SUCCESS {
		t.Fatalf("unexpected status: %s", r.Status)
	}
	if r.Stdout != "PIYO\n" {
		t.Fatalf("unexpected output: %s", r.Stdout)
	}
}
