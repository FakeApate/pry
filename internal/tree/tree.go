package tree

import (
	"net/url"
	"path"
	"sort"
	"strings"
	"time"
)

// Finding is the minimal input needed to build a tree node.
type Finding struct {
	URL           string
	ContentType   string
	ContentLength int64
	Category      string
	Interest      int
	LastModified  *time.Time
}

// Node represents a file or directory in the scanned tree.
type Node struct {
	Name      string  // segment name: "docs", "report.pdf"
	Path      string  // full relative path from root: "/docs/report.pdf"
	IsDir     bool    // true for directories
	Size      int64   // files: content_length; dirs: sum of all descendants
	FileCount int     // dirs: recursive count of file descendants
	Children  []*Node // sorted: dirs first (alpha), then files (alpha)

	// File-only fields
	ContentType string
	Category    string
	Interest    int
	LastModified *time.Time
}

// Build constructs a tree from a flat list of findings.
// rootURL is the scan's base URL; finding URLs are resolved relative to it.
func Build(rootURL string, findings []Finding) *Node {
	base := extractBasePath(rootURL)

	root := &Node{
		Name:  path.Base(base),
		Path:  "/",
		IsDir: true,
	}
	if root.Name == "" || root.Name == "." || root.Name == "/" {
		root.Name = "/"
	}

	for _, f := range findings {
		rel := relativePath(f.URL, base)
		if rel == "" {
			continue
		}
		insertFinding(root, rel, &f)
	}

	sortChildren(root)
	computeRollups(root)
	return root
}

// extractBasePath returns the path component of a URL, with trailing slash.
func extractBasePath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "/"
	}
	p := u.Path
	if !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return p
}

// relativePath strips the base path from a finding URL, returning the relative path.
func relativePath(findingURL, basePath string) string {
	u, err := url.Parse(findingURL)
	if err != nil {
		return ""
	}
	p := u.Path
	if after, ok := strings.CutPrefix(p, basePath); ok {
		return after
	}
	// If base doesn't match, try without trailing slash
	trimmed := strings.TrimRight(basePath, "/")
	if after, ok := strings.CutPrefix(p, trimmed); ok {
		return strings.TrimPrefix(after, "/")
	}
	return p
}

// insertFinding walks/creates the directory path and inserts a file node at the leaf.
func insertFinding(root *Node, relPath string, f *Finding) {
	parts := strings.Split(relPath, "/")
	current := root

	// Walk/create intermediate directories
	for i := 0; i < len(parts)-1; i++ {
		seg := parts[i]
		if seg == "" {
			continue
		}
		child := findChild(current, seg)
		if child == nil {
			child = &Node{
				Name:  seg,
				Path:  "/" + strings.Join(parts[:i+1], "/") + "/",
				IsDir: true,
			}
			current.Children = append(current.Children, child)
		}
		current = child
	}

	// Insert file node
	fileName := parts[len(parts)-1]
	if fileName == "" {
		return // trailing slash = directory, already handled
	}
	fileNode := &Node{
		Name:         fileName,
		Path:         "/" + relPath,
		IsDir:        false,
		Size:         f.ContentLength,
		ContentType:  f.ContentType,
		Category:     f.Category,
		Interest:     f.Interest,
		LastModified: f.LastModified,
	}
	current.Children = append(current.Children, fileNode)
}

func findChild(parent *Node, name string) *Node {
	for _, c := range parent.Children {
		if c.IsDir && c.Name == name {
			return c
		}
	}
	return nil
}

// sortChildren recursively sorts: dirs first alphabetically, then files alphabetically.
func sortChildren(n *Node) {
	if !n.IsDir {
		return
	}
	sort.SliceStable(n.Children, func(i, j int) bool {
		a, b := n.Children[i], n.Children[j]
		if a.IsDir != b.IsDir {
			return a.IsDir // dirs before files
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	for _, c := range n.Children {
		sortChildren(c)
	}
}

// computeRollups walks the tree bottom-up to compute Size and FileCount for directories.
func computeRollups(n *Node) {
	if !n.IsDir {
		return
	}
	n.Size = 0
	n.FileCount = 0
	for _, c := range n.Children {
		computeRollups(c)
		n.Size += c.Size
		if c.IsDir {
			n.FileCount += c.FileCount
		} else {
			n.FileCount++
		}
	}
}

// SortBy re-sorts all directory children by the given mode.
type SortMode int

const (
	SortByName     SortMode = iota
	SortBySize              // largest first
	SortByInterest          // highest first
)

// Resort re-sorts the entire tree by the given mode.
func Resort(n *Node, mode SortMode) {
	if !n.IsDir {
		return
	}
	sort.SliceStable(n.Children, func(i, j int) bool {
		a, b := n.Children[i], n.Children[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		switch mode {
		case SortBySize:
			return a.Size > b.Size
		case SortByInterest:
			return maxInterest(a) > maxInterest(b)
		default: // SortByName
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
	})
	for _, c := range n.Children {
		Resort(c, mode)
	}
}

// maxInterest returns the maximum interest score in a subtree.
func maxInterest(n *Node) int {
	if !n.IsDir {
		return n.Interest
	}
	best := 0
	for _, c := range n.Children {
		if v := maxInterest(c); v > best {
			best = v
		}
	}
	return best
}
