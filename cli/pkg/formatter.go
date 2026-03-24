package pkg

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// OutputFormat represents the output format type
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
	FormatQuiet OutputFormat = "quiet"
)

// ParseFormat parses format from string
func ParseFormat(s string) OutputFormat {
	switch strings.ToLower(s) {
	case "json":
		return FormatJSON
	case "csv":
		return FormatCSV
	case "quiet":
		return FormatQuiet
	default:
		return FormatTable
	}
}

// TableWriter writes data in table format
type TableWriter struct {
	w       io.Writer
	rows    [][]string
	maxCols int
}

// NewTableWriter creates a new table writer
func NewTableWriter(w io.Writer) *TableWriter {
	return &TableWriter{
		w:    w,
		rows: make([][]string, 0),
	}
}

// AddRow adds a row to the table
func (tw *TableWriter) AddRow(cols ...string) {
	if len(cols) > tw.maxCols {
		tw.maxCols = len(cols)
	}
	tw.rows = append(tw.rows, cols)
}

// Render renders the table
func (tw *TableWriter) Render() error {
	if len(tw.rows) == 0 {
		return nil
	}

	// Calculate column widths
	widths := make([]int, tw.maxCols)
	for _, row := range tw.rows {
		for i, col := range row {
			if len(col) > widths[i] {
				widths[i] = len(col)
			}
		}
	}

	// Render rows
	for i, row := range tw.rows {
		for j, col := range row {
			if j > 0 {
				fmt.Fprint(tw.w, "  ")
			}
			fmt.Fprintf(tw.w, "%-*s", widths[j], col)
		}
		fmt.Fprintln(tw.w)

		// Add separator after header
		if i == 0 {
			for j, width := range widths {
				if j > 0 {
					fmt.Fprint(tw.w, "  ")
				}
				fmt.Fprint(tw.w, strings.Repeat("-", width))
			}
			fmt.Fprintln(tw.w)
		}
	}

	return nil
}

// WriteJSON writes data in JSON format
func WriteJSON(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// WriteCSV writes data in CSV format
func WriteCSV(w io.Writer, headers []string, rows [][]string) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(headers); err != nil {
		return err
	}

	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// WriteQuiet writes minimal output (values only, one per line)
func WriteQuiet(w io.Writer, values []string) error {
	for _, v := range values {
		fmt.Fprintln(w, v)
	}
	return nil
}

// HasColor checks if output should have colors
func HasColor() bool {
	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check if stdout is a terminal
	// For simplicity, assume colors are supported
	return true
}

// Color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
)

// Colorize returns colored text if colors are enabled
func Colorize(text string, color string, enableColors bool) string {
	if !enableColors {
		return text
	}
	return color + text + ColorReset
}
