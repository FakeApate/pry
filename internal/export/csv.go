package export

import (
	"encoding/csv"
	"fmt"
	"io"
)

// CSVExporter writes scan findings as a flat CSV file.
type CSVExporter struct{}

func (CSVExporter) Export(w io.Writer, data ScanData) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	if err := cw.Write([]string{"URL", "Category", "ContentType", "Size", "InterestScore", "Tags", "LastModified"}); err != nil {
		return err
	}

	for _, f := range data.Findings {
		lastMod := ""
		if f.LastModified != nil {
			lastMod = f.LastModified.Format("2006-01-02T15:04:05Z")
		}
		if err := cw.Write([]string{
			f.URL,
			f.Category,
			f.ContentType,
			fmt.Sprintf("%d", f.ContentLength),
			fmt.Sprintf("%d", f.InterestScore),
			f.Tags,
			lastMod,
		}); err != nil {
			return err
		}
	}

	return cw.Error()
}
