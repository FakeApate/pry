package export

import (
	"io"
	"time"

	"github.com/fakeapate/pry/internal/tree"
)

// Finding holds the flat data for a single finding.
type Finding struct {
	URL           string
	ContentType   string
	ContentLength int64
	Category      string
	InterestScore int
	Tags          string
	LastModified  *time.Time
}

// ScanData holds everything needed to export a scan.
type ScanData struct {
	ScanID   string
	URL      string
	ScanDate time.Time
	Tree     *tree.Node
	Findings []Finding
	Total    int
}

// Exporter writes scan data to a writer in a specific format.
type Exporter interface {
	Export(w io.Writer, data ScanData) error
}
