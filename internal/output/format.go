package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// Provenance classifies the source of a data field.
type Provenance string

const (
	Observed Provenance = "observed" // Live from MySQL query.
	Declared Provenance = "declared" // From config or annotation.
	Inferred Provenance = "inferred" // Computed or derived.
	Unknown  Provenance = "unknown"  // Source unclear or stale.
)

// Result is the standard output envelope for all commands.
type Result struct {
	Data       any                   `json:"data"`
	Provenance map[string]Provenance `json:"provenance,omitempty"`
}

// Table is a simple tabular data structure for table-format output.
type Table struct {
	Headers []string
	Rows    [][]string
}

// Render writes output in the specified format.
func Render(format string, result Result, table *Table) error {
	return FRender(os.Stdout, format, result, table)
}

// FRender writes output to a specific writer.
func FRender(w io.Writer, format string, result Result, table *Table) error {
	switch strings.ToLower(format) {
	case "json":
		return renderJSON(w, result)
	case "table", "":
		if table != nil {
			return renderTable(w, table)
		}
		return renderJSON(w, result)
	default:
		return fmt.Errorf("unsupported format %q, use json or table", format)
	}
}

func renderJSON(w io.Writer, result Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func renderTable(w io.Writer, t *Table) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if len(t.Headers) > 0 {
		_, _ = fmt.Fprintln(tw, strings.Join(t.Headers, "\t"))
	}
	for _, row := range t.Rows {
		_, _ = fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	return tw.Flush()
}
