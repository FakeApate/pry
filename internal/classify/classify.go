package classify

import (
	"net/url"
	"path"
	"strings"
)

// Result holds the classification output for a single finding.
type Result struct {
	Category      string   // Primary category (one of the Cat* constants)
	Tags          []string // Secondary labels: "sensitive", "backup", "log", etc.
	InterestScore int      // 0–100
}

// Classify determines the category, tags, and interest score for a finding
// based on its URL, content type, and size. It is a pure function with no
// side effects.
func Classify(rawURL string, contentType string, contentLength int64) Result {
	ext := extractExt(rawURL)
	filename := extractFilename(rawURL)
	filenameLower := strings.ToLower(filename)

	cat := categorize(ext, contentType)
	tags := computeTags(ext, filenameLower)
	score := computeScore(cat, tags, ext, filenameLower, contentLength)

	return Result{
		Category:      cat,
		Tags:          tags,
		InterestScore: score,
	}
}

// categorize determines the primary category, preferring extension over MIME.
func categorize(ext string, contentType string) string {
	if cat, ok := extCategory[ext]; ok {
		return cat
	}
	ct := strings.ToLower(contentType)
	for _, m := range mimeCategory {
		if strings.HasPrefix(ct, m.prefix) {
			return m.category
		}
	}
	return CatOther
}

// computeTags builds the secondary tag list from extension and filename patterns.
func computeTags(ext string, filenameLower string) []string {
	var tags []string

	if sensitiveExts[ext] {
		tags = append(tags, "sensitive")
	}
	for _, p := range sensitivePatterns {
		if strings.Contains(filenameLower, p) {
			if !hasTag(tags, "sensitive") {
				tags = append(tags, "sensitive")
			}
			break
		}
	}

	for _, p := range backupPatterns {
		if strings.Contains(filenameLower, p) {
			tags = append(tags, "backup")
			break
		}
	}

	if ext == ".log" || strings.Contains(filenameLower, "error_log") || strings.Contains(filenameLower, "access_log") {
		tags = append(tags, "log")
	}

	return tags
}

// computeScore calculates the interest score (0–100) from category, tags, and signals.
func computeScore(cat string, tags []string, ext string, filenameLower string, contentLength int64) int {
	score := baseCategoryScore[cat]

	if hasTag(tags, "sensitive") {
		score += 40
	}
	if hasTag(tags, "backup") {
		score += 20
	}
	if hasTag(tags, "log") {
		score += 10
	}
	if rareExts[ext] {
		score += 15
	}

	// Size bonuses
	const mb100 = 100 * 1024 * 1024
	const gb1 = 1024 * 1024 * 1024
	if contentLength > gb1 {
		score += 15
	} else if contentLength > mb100 {
		score += 10
	}

	if score > 100 {
		score = 100
	}
	return score
}

// extractExt returns the lowercase file extension from a URL path.
func extractExt(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	ext := strings.ToLower(path.Ext(u.Path))
	return ext
}

// extractFilename returns the last path segment from a URL.
func extractFilename(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return path.Base(u.Path)
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}
