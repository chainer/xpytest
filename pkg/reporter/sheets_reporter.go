package reporter

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

type sheetsReporter struct {
	client        *http.Client
	spreadsheetID string
	values        [][]interface{}
}

// NewSheetsReporterWithCredential creates a reporter to store logs to a
// spreadsheet with a JSON credential.
func NewSheetsReporterWithCredential(
	ctx context.Context, cred, spreadsheetID string,
) (Reporter, error) {
	if buf, err := ioutil.ReadFile(cred); err != nil {
		return nil, err
	} else if conf, err := google.JWTConfigFromJSON(
		buf, sheets.SpreadsheetsScope); err != nil {
		return nil, err
	} else {
		return &sheetsReporter{
			client:        conf.Client(oauth2.NoContext),
			spreadsheetID: spreadsheetID,
		}, nil
	}
}

// NewSheetsReporter creates a reporter to store logs to a spreadsheet with
// a default credential.
func NewSheetsReporter(
	ctx context.Context, spreadsheetID string,
) (Reporter, error) {
	client, err := google.DefaultClient(ctx, sheets.SpreadsheetsScope)
	if err != nil {
		return nil, err
	}
	return &sheetsReporter{
		client:        client,
		spreadsheetID: spreadsheetID,
	}, nil
}

func (r *sheetsReporter) Log(ctx context.Context, msg string) {
	if r.values == nil {
		r.values = [][]interface{}{}
	}
	r.values = append(r.values, []interface{}{msg})
}

func (r *sheetsReporter) Flush(ctx context.Context) error {
	svc, err := sheets.New(r.client)
	if err != nil {
		return fmt.Errorf("failed to get sheets client: %s", err)
	}
	_, err = svc.Spreadsheets.Values.Append(
		r.spreadsheetID, "A1", &sheets.ValueRange{Values: r.values},
	).ValueInputOption("RAW").Do()
	if err != nil {
		return fmt.Errorf("failed to append rows: %s", err)
	}
	r.values = nil
	return nil
}
