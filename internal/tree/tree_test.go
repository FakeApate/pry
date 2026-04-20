package tree

import (
	"testing"
	"time"
)

func TestBuildEmpty(t *testing.T) {
	root := Build("https://example.com/files/", nil)
	if !root.IsDir {
		t.Error("root should be a directory")
	}
	if root.FileCount != 0 {
		t.Errorf("expected 0 files, got %d", root.FileCount)
	}
	if root.Size != 0 {
		t.Errorf("expected size 0, got %d", root.Size)
	}
}

func TestBuildSingleFile(t *testing.T) {
	findings := []Finding{
		{URL: "https://example.com/files/report.pdf", ContentLength: 2048, Category: "document"},
	}
	root := Build("https://example.com/files/", findings)
	if root.FileCount != 1 {
		t.Errorf("expected 1 file, got %d", root.FileCount)
	}
	if root.Size != 2048 {
		t.Errorf("expected size 2048, got %d", root.Size)
	}
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(root.Children))
	}
	f := root.Children[0]
	if f.Name != "report.pdf" {
		t.Errorf("expected name report.pdf, got %q", f.Name)
	}
	if f.IsDir {
		t.Error("file should not be a directory")
	}
	if f.Category != "document" {
		t.Errorf("expected category document, got %q", f.Category)
	}
}

func TestBuildNestedDirs(t *testing.T) {
	findings := []Finding{
		{URL: "https://example.com/files/docs/a.pdf", ContentLength: 100},
		{URL: "https://example.com/files/docs/b.pdf", ContentLength: 200},
		{URL: "https://example.com/files/src/main.go", ContentLength: 50},
		{URL: "https://example.com/files/readme.txt", ContentLength: 10},
	}
	root := Build("https://example.com/files/", findings)

	if root.FileCount != 4 {
		t.Errorf("root file count: expected 4, got %d", root.FileCount)
	}
	if root.Size != 360 {
		t.Errorf("root size: expected 360, got %d", root.Size)
	}

	// Check directory structure
	if len(root.Children) != 3 {
		t.Fatalf("expected 3 children (docs/, src/, readme.txt), got %d", len(root.Children))
	}

	// Dirs should come first, sorted alphabetically
	if root.Children[0].Name != "docs" || !root.Children[0].IsDir {
		t.Errorf("first child should be docs/, got %q (dir=%v)", root.Children[0].Name, root.Children[0].IsDir)
	}
	if root.Children[1].Name != "src" || !root.Children[1].IsDir {
		t.Errorf("second child should be src/, got %q", root.Children[1].Name)
	}
	if root.Children[2].Name != "readme.txt" || root.Children[2].IsDir {
		t.Errorf("third child should be readme.txt file, got %q (dir=%v)", root.Children[2].Name, root.Children[2].IsDir)
	}

	// Check docs/ rollup
	docs := root.Children[0]
	if docs.FileCount != 2 {
		t.Errorf("docs file count: expected 2, got %d", docs.FileCount)
	}
	if docs.Size != 300 {
		t.Errorf("docs size: expected 300, got %d", docs.Size)
	}
}

func TestBuildDeepNesting(t *testing.T) {
	findings := []Finding{
		{URL: "https://example.com/a/b/c/d/file.txt", ContentLength: 42},
	}
	root := Build("https://example.com/", findings)

	// Walk down a/b/c/d/file.txt
	current := root
	for _, name := range []string{"a", "b", "c", "d"} {
		found := false
		for _, c := range current.Children {
			if c.Name == name && c.IsDir {
				current = c
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing directory %q", name)
		}
	}

	if len(current.Children) != 1 || current.Children[0].Name != "file.txt" {
		t.Error("expected file.txt at the leaf")
	}

	// All intermediate dirs should roll up
	if root.FileCount != 1 {
		t.Errorf("root file count: expected 1, got %d", root.FileCount)
	}
	if root.Size != 42 {
		t.Errorf("root size: expected 42, got %d", root.Size)
	}
}

func TestBuildRollups(t *testing.T) {
	findings := []Finding{
		{URL: "https://example.com/dir/small.txt", ContentLength: 100},
		{URL: "https://example.com/dir/big.bin", ContentLength: 1000000},
		{URL: "https://example.com/dir/sub/a.txt", ContentLength: 50},
		{URL: "https://example.com/dir/sub/b.txt", ContentLength: 50},
		{URL: "https://example.com/other.txt", ContentLength: 1},
	}
	root := Build("https://example.com/", findings)

	if root.FileCount != 5 {
		t.Errorf("root file count: expected 5, got %d", root.FileCount)
	}
	if root.Size != 1000201 {
		t.Errorf("root size: expected 1000201, got %d", root.Size)
	}

	// Find dir/
	var dir *Node
	for _, c := range root.Children {
		if c.Name == "dir" {
			dir = c
			break
		}
	}
	if dir == nil {
		t.Fatal("missing dir/")
	}
	if dir.FileCount != 4 {
		t.Errorf("dir file count: expected 4, got %d", dir.FileCount)
	}
	if dir.Size != 1000200 {
		t.Errorf("dir size: expected 1000200, got %d", dir.Size)
	}
}

func TestBuildSortOrder(t *testing.T) {
	findings := []Finding{
		{URL: "https://example.com/zebra.txt", ContentLength: 1},
		{URL: "https://example.com/beta/file.txt", ContentLength: 1},
		{URL: "https://example.com/alpha/file.txt", ContentLength: 1},
		{URL: "https://example.com/apple.txt", ContentLength: 1},
	}
	root := Build("https://example.com/", findings)

	// Dirs first alphabetically, then files alphabetically
	names := make([]string, len(root.Children))
	for i, c := range root.Children {
		names[i] = c.Name
	}
	expected := []string{"alpha", "beta", "apple.txt", "zebra.txt"}
	for i, want := range expected {
		if i >= len(names) || names[i] != want {
			t.Errorf("child[%d]: expected %q, got %v", i, want, names)
			break
		}
	}
}

func TestResortBySize(t *testing.T) {
	findings := []Finding{
		{URL: "https://example.com/small.txt", ContentLength: 10},
		{URL: "https://example.com/big.txt", ContentLength: 10000},
		{URL: "https://example.com/medium.txt", ContentLength: 500},
	}
	root := Build("https://example.com/", findings)
	Resort(root, SortBySize)

	if root.Children[0].Name != "big.txt" {
		t.Errorf("expected biggest first, got %q", root.Children[0].Name)
	}
	if root.Children[1].Name != "medium.txt" {
		t.Errorf("expected medium second, got %q", root.Children[1].Name)
	}
}

func TestResortByInterest(t *testing.T) {
	findings := []Finding{
		{URL: "https://example.com/boring.txt", Interest: 5},
		{URL: "https://example.com/interesting.sql", Interest: 55},
		{URL: "https://example.com/meh.pdf", Interest: 10},
	}
	root := Build("https://example.com/", findings)
	Resort(root, SortByInterest)

	if root.Children[0].Name != "interesting.sql" {
		t.Errorf("expected highest interest first, got %q", root.Children[0].Name)
	}
}

func TestBuildURLEdgeCases(t *testing.T) {
	// Base URL without trailing slash
	findings := []Finding{
		{URL: "https://example.com/files/test.txt", ContentLength: 1},
	}
	root := Build("https://example.com/files", findings)
	if root.FileCount != 1 {
		t.Errorf("expected 1 file, got %d", root.FileCount)
	}

	// Encoded characters
	findings2 := []Finding{
		{URL: "https://example.com/my%20files/test.txt", ContentLength: 1},
	}
	root2 := Build("https://example.com/my%20files/", findings2)
	if root2.FileCount != 1 {
		t.Errorf("encoded: expected 1 file, got %d", root2.FileCount)
	}
}

func TestBuildPreservesMetadata(t *testing.T) {
	mod := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	findings := []Finding{
		{
			URL:           "https://example.com/data.sql",
			ContentType:   "application/sql",
			ContentLength: 8300000,
			Category:      "database",
			Interest:      55,
			LastModified:  &mod,
		},
	}
	root := Build("https://example.com/", findings)
	f := root.Children[0]

	if f.ContentType != "application/sql" {
		t.Errorf("content type: %q", f.ContentType)
	}
	if f.Category != "database" {
		t.Errorf("category: %q", f.Category)
	}
	if f.Interest != 55 {
		t.Errorf("interest: %d", f.Interest)
	}
	if f.LastModified == nil || !f.LastModified.Equal(mod) {
		t.Errorf("last modified: %v", f.LastModified)
	}
}

func TestRootName(t *testing.T) {
	root := Build("https://example.com/files/", nil)
	if root.Name != "files" {
		t.Errorf("expected root name 'files', got %q", root.Name)
	}

	root2 := Build("https://example.com/", nil)
	if root2.Name != "/" {
		t.Errorf("expected root name '/', got %q", root2.Name)
	}
}
