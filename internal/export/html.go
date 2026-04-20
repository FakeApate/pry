package export

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"time"

	"github.com/fakeapate/pry/internal/tree"
)

//go:embed html_template.html
var htmlTemplateFS embed.FS

// HTMLExporter writes a self-contained interactive HTML file.
type HTMLExporter struct{}

type htmlData struct {
	ScanID   string
	URL      string
	ScanDate string
	Total    int
	TreeJSON template.JS
	Stats    htmlStats
}

type htmlStats struct {
	FileCount  int
	TotalSize  string
	Categories map[string]int
}

type htmlNode struct {
	Name         string      `json:"name"`
	Path         string      `json:"path"`
	IsDir        bool        `json:"dir"`
	Size         int64       `json:"size"`
	FileCount    int         `json:"fc,omitempty"`
	ContentType  string      `json:"ct,omitempty"`
	Category     string      `json:"cat,omitempty"`
	Interest     int         `json:"int,omitempty"`
	LastModified string      `json:"mod,omitempty"`
	Children     []*htmlNode `json:"ch,omitempty"`
}

func toHTMLNode(n *tree.Node) *htmlNode {
	hn := &htmlNode{
		Name:      n.Name,
		Path:      n.Path,
		IsDir:     n.IsDir,
		Size:      n.Size,
		FileCount: n.FileCount,
		ContentType: n.ContentType,
		Category:  n.Category,
		Interest:  n.Interest,
	}
	if n.LastModified != nil {
		hn.LastModified = n.LastModified.Format("2006-01-02")
	}
	for _, c := range n.Children {
		hn.Children = append(hn.Children, toHTMLNode(c))
	}
	return hn
}

func (HTMLExporter) Export(w io.Writer, data ScanData) error {
	tmpl, err := template.ParseFS(htmlTemplateFS, "html_template.html")
	if err != nil {
		return err
	}

	treeNode := toHTMLNode(data.Tree)
	treeBytes, err := json.Marshal(treeNode)
	if err != nil {
		return err
	}

	cats := map[string]int{}
	for _, f := range data.Findings {
		if f.Category != "" {
			cats[f.Category]++
		}
	}

	hd := htmlData{
		ScanID:   data.ScanID,
		URL:      data.URL,
		ScanDate: data.ScanDate.Format(time.RFC3339),
		Total:    data.Total,
		TreeJSON: template.JS(treeBytes),
		Stats: htmlStats{
			FileCount:  data.Tree.FileCount,
			TotalSize:  humanBytes(data.Tree.Size),
			Categories: cats,
		},
	}

	return tmpl.Execute(w, hd)
}

func humanBytes(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	case n < 1024*1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	default:
		return fmt.Sprintf("%.1f GB", float64(n)/(1024*1024*1024))
	}
}
