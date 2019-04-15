package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/chainer/xpytest/pkg/reporter"
)

var credential = flag.String(
	"credential", "", "JSON credential file for Google")
var spreadsheetID = flag.String("spreadsheet_id", "", "spreadsheet ID to edit")

func main() {
	ctx := context.Background()
	r, err := func() (reporter.Reporter, error) {
		if *credential != "" {
			return reporter.NewSheetsReporterWithCredential(
				ctx, *credential, *spreadsheetID)
		}
		return reporter.NewSheetsReporter(ctx, *spreadsheetID)
	}()
	if err != nil {
		panic(fmt.Sprintf("failed to create sheets reporter: %s", err))
	}

	for i := 0; i < 10; i++ {
		r.Log(ctx, fmt.Sprintf("%s] test log %d", time.Now().String(), i))
	}

	if err := r.Flush(ctx); err != nil {
		panic(fmt.Sprintf("failed to flush sheets reporter: %s", err))
	}
}
