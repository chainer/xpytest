package pytest

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"

	xpytest_proto "github.com/chainer/xpytest/proto"
)

// Execute executes a command.
func Execute(
	ctx context.Context, args []string, deadline time.Duration, env []string,
) (*xpytest_proto.TestResult, error) {
	startTime := time.Now()

	type executeResult struct {
		testResult *xpytest_proto.TestResult
		err        error
	}
	resultChan := make(chan *executeResult, 2)
	done := make(chan struct{}, 1)

	temporaryResult := &xpytest_proto.TestResult{}
	go func() {
		err := executeInternal(
			ctx, args, deadline, env, temporaryResult)
		resultChan <- &executeResult{testResult: temporaryResult, err: err}
		close(done)
	}()

	go func() {
		select {
		case <-done:
		case <-time.After(deadline + 5*time.Second):
			r := proto.Clone(temporaryResult).(*xpytest_proto.TestResult)
			r.Status = xpytest_proto.TestResult_TIMEOUT
			resultChan <- &executeResult{testResult: r, err: nil}
			fmt.Fprintf(os.Stderr, "[ERROR] command is hung up: %s\n",
				strings.Join(args, " "))
		}
	}()

	result := <-resultChan
	if result.err != nil {
		return nil, result.err
	}

	result.testResult.Time =
		float32(time.Now().Sub(startTime)) / float32(time.Second)
	return result.testResult, nil
}

func executeInternal(
	ctx context.Context, args []string, deadline time.Duration, env []string,
	result *xpytest_proto.TestResult,
) error {
	// Prepare a Cmd object.
	if len(args) == 0 {
		return fmt.Errorf("# of args must be larger than 0")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	// Open pipes.
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %s", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %s", err)
	}

	// Set environment variables.
	if env == nil {
		env = []string{}
	}
	env = append(env, os.Environ()...)
	cmd.Env = env

	// Start the command.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %s", err)
	}

	// Prepare a wait group to maintain threads.
	wg := sync.WaitGroup{}
	async := func(f func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f()
		}()
	}

	// Run I/O threads.
	readAll := func(pipe io.ReadCloser, out *string) {
		s := bufio.NewReaderSize(pipe, 128)
		for {
			line, err := s.ReadSlice('\n')
			if err == io.EOF {
				break
			} else if err != nil && err != bufio.ErrBufferFull {
				if err.Error() != "read |0: file already closed" {
					fmt.Fprintf(os.Stderr,
						"[ERROR] failed to read from pipe: %s\n", err)
				}
				break
			}
			*out += string(line)
		}
		pipe.Close()
	}
	async(func() { readAll(stdoutPipe, &result.Stdout) })
	async(func() { readAll(stderrPipe, &result.Stderr) })

	// Run timer thread.
	var timeout bool
	cmdIsDone := make(chan struct{}, 1)
	async(func() {
		select {
		case <-cmdIsDone:
		case <-time.After(deadline):
			timeout = true
			cmd.Process.Kill()
		}
	})

	// Wait for the command.
	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "[DEBUG] failed to wait a command: %s: %s\n",
			strings.Join(args, " "), err)
		cmd.Process.Kill()
	}
	close(cmdIsDone)
	wg.Wait()

	// Get the last line.
	if timeout {
		result.Status = xpytest_proto.TestResult_TIMEOUT
	} else if cmd.ProcessState.Success() {
		result.Status = xpytest_proto.TestResult_SUCCESS
	} else {
		result.Status = xpytest_proto.TestResult_FAILED
	}

	return nil
}
