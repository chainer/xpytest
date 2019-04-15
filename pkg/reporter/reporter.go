package reporter

import "context"

// Reporter reports lines.
type Reporter interface {
	// Log appends a line to the report buffer.  Reporter will not write lines
	// until Reporter.Flush is called.
	Log(context.Context, string)

	// Flush flushes the report buffer.
	Flush(context.Context) error
}
