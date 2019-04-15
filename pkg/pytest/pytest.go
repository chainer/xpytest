package pytest

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	xpytest_proto "github.com/chainer/xpytest/proto"
)

// Pytest represents one pytest execution.
type Pytest struct {
	PythonCmd        string
	MarkerExpression string
	Xdist            int
	Files            []string
	Executor         func(
		context.Context, []string, time.Duration, []string,
	) (*xpytest_proto.TestResult, error)
	Retry    int
	Env      []string
	Deadline time.Duration
}

// NewPytest creates a new Pytest object.
func NewPytest(pythonCmd string) *Pytest {
	return &Pytest{PythonCmd: pythonCmd, Executor: Execute}
}

// Execute builds pytest parameters and runs pytest.
func (p *Pytest) Execute(
	ctx context.Context,
) (*Result, error) {
	var finalResult *Result
	for trial := 0; trial == 0 || trial < p.Retry; trial++ {
		pr, err := p.execute(ctx)
		if err != nil {
			return nil, err
		}
		if trial == 0 {
			finalResult = pr
		} else if pr.Status == xpytest_proto.TestResult_SUCCESS {
			finalResult.Status = xpytest_proto.TestResult_FLAKY
		}
		finalResult.trial = trial
		if finalResult.Status != xpytest_proto.TestResult_FAILED {
			break
		}
	}
	return finalResult, nil
}

func (p *Pytest) execute(
	ctx context.Context,
) (*Result, error) {
	// Build command-line arguments.
	args := []string{p.PythonCmd, "-m", "pytest"}
	if p.MarkerExpression != "" {
		args = append(args, "-m", p.MarkerExpression)
	}
	if p.Xdist > 0 {
		args = append(args, "-n", fmt.Sprintf("%d", p.Xdist))
	}
	if len(p.Files) == 0 {
		return nil, errors.New("Pytest.Files must not be empty")
	}
	args = append(args, p.Files...)

	// Check deadline.
	deadline := p.Deadline
	if deadline <= 0 {
		return nil, fmt.Errorf("Pytest.Deadline must be postiive value")
	}

	// Execute pytest.
	r, err := p.Executor(ctx, args, deadline, p.Env)
	if err != nil {
		return nil, err
	}
	return newPytestResult(p, r), nil
}

// Result represents a pytest execution result.
type Result struct {
	Status   xpytest_proto.TestResult_Status
	Name     string
	xdist    int
	trial    int
	duration float32
	summary  string
	stdout   string
	stderr   string
}

func newPytestResult(p *Pytest, tr *xpytest_proto.TestResult) *Result {
	r := &Result{}
	if len(p.Files) > 0 {
		r.Name = p.Files[0]
	}
	r.Status = tr.GetStatus()
	result := ""
	if r.Status != xpytest_proto.TestResult_TIMEOUT {
		lines := strings.Split(strings.TrimSpace(tr.Stdout), "\n")
		lastLine := lines[len(lines)-1]
		if strings.HasPrefix(lastLine, "=") {
			result = strings.Trim(lastLine, "= ")
		} else {
			result = fmt.Sprintf("%s; %.0f seconds", r.Status, r.duration)
			r.Status = xpytest_proto.TestResult_INTERNAL
		}
		// pytest's message for no tests.
		if regexp.MustCompile(
			`^\d+ deselected in \d+(\.\d+)? seconds$`).MatchString(result) {
			r.Status = xpytest_proto.TestResult_SUCCESS
		}
		// pytest-xdist's message for no tests.
		if regexp.MustCompile(
			`^no tests ran in \d+(\.\d+)? seconds$`).MatchString(result) {
			r.Status = xpytest_proto.TestResult_SUCCESS
		}
	}
	r.xdist = p.Xdist
	r.duration = tr.GetTime()
	r.summary = func() string {
		if r.Status == xpytest_proto.TestResult_TIMEOUT {
			return fmt.Sprintf("%.0f seconds", r.duration)
		}
		return fmt.Sprintf("%s", result)
	}()
	shorten := func(s string) string {
		ss := strings.Split(s, "\n")
		if len(ss) > 500 {
			output := ss[0:250]
			output = append(output,
				fmt.Sprintf("...(%d lines skipped)...", len(ss)-500))
			output = append(output, ss[len(ss)-250:]...)
			return strings.Join(output, "\n")
		}
		return s
	}
	r.stdout = shorten(tr.Stdout)
	r.stderr = shorten(tr.Stderr)
	return r
}

// Summary returns a one-line summary of the test result (e.g.,
// "[SUCCESS] test_foo.py (123 passed in 4.56 seconds)").
func (r *Result) Summary() string {
	ss := []string{}
	if r.summary != "" {
		ss = append(ss, r.summary)
	}
	if r.xdist > 0 {
		ss = append(ss, fmt.Sprintf("%d procs", r.xdist))
	}
	if r.trial > 0 {
		ss = append(ss, fmt.Sprintf("%d trials", r.trial+1))
	}
	s := strings.Join(ss, " * ")
	if s != "" {
		s = " (" + s + ")"
	}
	return fmt.Sprintf("[%s] %s%s", r.Status, r.Name, s)
}

// Output returns the test result.  This returns outputs from STDOUT/STDERR in
// addition to a one-line summary returned by Summary.
func (r *Result) Output() string {
	if r.Status == xpytest_proto.TestResult_SUCCESS {
		return strings.TrimSpace(r.Summary() + "\n" + r.stderr)
	}
	return strings.TrimSpace(r.Summary() + "\n" +
		strings.TrimSpace(r.stdout+"\n"+r.stderr))
}
