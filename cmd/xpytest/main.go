package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	xpytest_proto "github.com/chainer/xpytest/proto"

	"github.com/chainer/xpytest/pkg/pytest"
	"github.com/chainer/xpytest/pkg/reporter"
	"github.com/chainer/xpytest/pkg/xpytest"
)

var python = flag.String("python", "python3", "python command")
var markerExpression = flag.String("m", "not slow", "pytest marker expression")
var retry = flag.Int("retry", 2, "number of retries")
var credential = flag.String(
	"credential", "", "JSON credential file for Google")
var spreadsheetID = flag.String("spreadsheet_id", "", "spreadsheet ID to edit")
var hint = flag.String("hint", "", "hint file")
var bucket = flag.Int("bucket", 1, "number of buckets")
var thread = flag.Int("thread", 0, "number of threads per bucket")
var reportName = flag.String("report_name", "", "name for reporter")

func main() {
	flag.Parse()
	ctx := context.Background()

	base := pytest.NewPytest(*python)
	base.MarkerExpression = *markerExpression
	base.Retry = *retry
	base.Deadline = time.Minute
	xt := xpytest.NewXpytest(base)

	r, err := func() (reporter.Reporter, error) {
		if *spreadsheetID == "" {
			return nil, nil
		}
		if *credential != "" {
			return reporter.NewSheetsReporterWithCredential(
				ctx, *credential, *spreadsheetID)
		}
		return reporter.NewSheetsReporter(ctx, *spreadsheetID)
	}()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize reporter: %s", err))
	}
	if r != nil {
		if *reportName != "" {
			r.Log(ctx, *reportName)
		} else {
			r.Log(ctx, fmt.Sprintf("Time: %s", time.Now()))
		}
	}

	for _, arg := range flag.Args() {
		if err := xt.AddTestsWithFilePattern(arg); err != nil {
			panic(fmt.Sprintf("failed to add tests: %s", err))
		}
	}

	if *hint != "" {
		if h, err := xpytest.LoadHintFile(*hint); err != nil {
			panic(fmt.Sprintf(
				"failed to read hint information from file: %s: %s",
				*hint, err))
		} else if err := xt.ApplyHint(h); err != nil {
			panic(fmt.Sprintf("failed to apply hint: %s", err))
		}
	}

	if err := xt.Execute(ctx, *bucket, *thread, r); err != nil {
		panic(fmt.Sprintf("failed to execute: %s", err))
	}

	if r != nil {
		fmt.Fprintf(os.Stderr, "[DEBUG] flushing reporter...\n")
		if err := r.Flush(ctx); err != nil {
			fmt.Fprintf(os.Stderr,
				"[ERROR] failed to flush reporter: %s\n", err)
		}
	}

	fmt.Printf("Overall status: %s\n", xt.Status)
	if xt.Status != xpytest_proto.TestResult_SUCCESS &&
		xt.Status != xpytest_proto.TestResult_FLAKY {
		os.Exit(1)
	}
}
