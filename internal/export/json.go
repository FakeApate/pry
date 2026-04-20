package export

import (
	"encoding/json"
	"io"
	"time"

	"github.com/fakeapate/pry/internal/tree"
)

// JSONExporter writes scan data as structured JSON with the tree hierarchy.
type JSONExporter struct{}

type jsonOutput struct {
	ScanID   string    `json:"scan_id"`
	URL      string    `json:"url"`
	ScanDate time.Time `json:"scan_date"`
	Total    int       `json:"total_findings"`
	Tree     *jsonNode `json:"tree"`
}

type jsonNode struct {
	Name         string      `json:"name"`
	Path         string      `json:"path"`
	IsDir        bool        `json:"is_dir"`
	Size         int64       `json:"size"`
	FileCount    int         `json:"file_count,omitempty"`
	ContentType  string      `json:"content_type,omitempty"`
	Category     string      `json:"category,omitempty"`
	Interest     int         `json:"interest,omitempty"`
	LastModified *time.Time  `json:"last_modified,omitempty"`
	Children     []*jsonNode `json:"children,omitempty"`
}

func toJSONNode(n *tree.Node) *jsonNode {
	jn := &jsonNode{
		Name:         n.Name,
		Path:         n.Path,
		IsDir:        n.IsDir,
		Size:         n.Size,
		FileCount:    n.FileCount,
		ContentType:  n.ContentType,
		Category:     n.Category,
		Interest:     n.Interest,
		LastModified: n.LastModified,
	}
	for _, c := range n.Children {
		jn.Children = append(jn.Children, toJSONNode(c))
	}
	return jn
}

func (JSONExporter) Export(w io.Writer, data ScanData) error {
	out := jsonOutput{
		ScanID:   data.ScanID,
		URL:      data.URL,
		ScanDate: data.ScanDate,
		Total:    data.Total,
		Tree:     toJSONNode(data.Tree),
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
