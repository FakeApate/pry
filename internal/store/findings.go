package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type FindingsFilter struct {
	ScanID       string
	ContentTypes []string
	Categories   []string
	MinSize      *int64
	MaxSize      *int64
	MinInterest  *int
	Query        *string
	SortBy       string
	SortOrder    string
	Page         int
	PageSize     int
}

// AllFindingsPageSize is the page size callers use to load an entire scan's
// findings in one shot (tree build, export). It is an upper bound rather than
// a soft cap; callers that hit it silently truncate the result.
const AllFindingsPageSize = 1_000_000

type Finding struct {
	URL           string
	ScanTime      time.Time
	ContentType   string
	ContentLength int64
	LastModified  *time.Time
	Category      string
	InterestScore int
	Tags          string
}

type FindingsResult struct {
	Findings []Finding
	Total    int
}

type FindingsStore struct {
	db *sql.DB
}

func NewFindingsStore(db *sql.DB) *FindingsStore {
	return &FindingsStore{db: db}
}

var allowedSortColumns = map[string]string{
	"url":            "url",
	"content_type":   "content_type",
	"content_length": "content_length",
	"last_modified":  "last_modified",
	"category":       "category",
	"interest_score": "interest_score",
}

// likeEscaper escapes the LIKE wildcards ('%' and '_') so search queries only
// match literal characters. Pairs with `ESCAPE '\'` in the generated SQL.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (s *FindingsStore) QueryFindings(ctx context.Context, f FindingsFilter) (FindingsResult, error) {
	var conditions []string
	var whereArgs []any

	conditions = append(conditions, "scan_id = ?")
	whereArgs = append(whereArgs, f.ScanID)

	if len(f.ContentTypes) > 0 {
		placeholders := make([]string, len(f.ContentTypes))
		for i, ct := range f.ContentTypes {
			placeholders[i] = "?"
			whereArgs = append(whereArgs, ct)
		}
		conditions = append(conditions, fmt.Sprintf("content_type IN (%s)", strings.Join(placeholders, ", ")))
	}
	if len(f.Categories) > 0 {
		placeholders := make([]string, len(f.Categories))
		for i, cat := range f.Categories {
			placeholders[i] = "?"
			whereArgs = append(whereArgs, cat)
		}
		conditions = append(conditions, fmt.Sprintf("category IN (%s)", strings.Join(placeholders, ", ")))
	}
	if f.MinInterest != nil {
		conditions = append(conditions, "interest_score >= ?")
		whereArgs = append(whereArgs, *f.MinInterest)
	}
	if f.MinSize != nil {
		conditions = append(conditions, "content_length >= ?")
		whereArgs = append(whereArgs, *f.MinSize)
	}
	if f.MaxSize != nil {
		conditions = append(conditions, "content_length <= ?")
		whereArgs = append(whereArgs, *f.MaxSize)
	}
	if f.Query != nil {
		conditions = append(conditions, `url LIKE ? ESCAPE '\'`)
		whereArgs = append(whereArgs, "%"+likeEscaper.Replace(*f.Query)+"%")
	}

	where := strings.Join(conditions, " AND ")

	sortCol := "url"
	if col, ok := allowedSortColumns[f.SortBy]; ok {
		sortCol = col
	}
	sortDir := "ASC"
	if strings.EqualFold(f.SortOrder, "desc") {
		sortDir = "DESC"
	}

	offset := (f.Page - 1) * f.PageSize

	query := fmt.Sprintf(
		`SELECT url, scan_time, content_type, content_length, last_modified, category, interest_score, tags, COUNT(*) OVER() AS total
		FROM scan_findings
		WHERE %s
		ORDER BY %s %s
		LIMIT ? OFFSET ?`,
		where, sortCol, sortDir,
	)
	pageArgs := append(append([]any{}, whereArgs...), f.PageSize, offset)

	rows, err := s.db.QueryContext(ctx, query, pageArgs...)
	if err != nil {
		return FindingsResult{}, err
	}
	defer rows.Close()

	var result FindingsResult
	for rows.Next() {
		var f Finding
		var scanTime string
		var lastMod sql.NullString
		var total int
		if err := rows.Scan(&f.URL, &scanTime, &f.ContentType, &f.ContentLength, &lastMod, &f.Category, &f.InterestScore, &f.Tags, &total); err != nil {
			return FindingsResult{}, err
		}
		f.ScanTime, _ = time.Parse(time.RFC3339, scanTime)
		if lastMod.Valid {
			t, err := time.Parse(time.RFC3339, lastMod.String)
			if err == nil {
				f.LastModified = &t
			}
		}
		result.Total = total
		result.Findings = append(result.Findings, f)
	}
	if err := rows.Err(); err != nil {
		return FindingsResult{}, err
	}

	// Window-function totals are 0 when no rows came back; re-run the count
	// with the WHERE-clause args only (no pagination args).
	if len(result.Findings) == 0 {
		var count int
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM scan_findings WHERE %s", where)
		if err := s.db.QueryRowContext(ctx, countQuery, whereArgs...).Scan(&count); err != nil {
			return FindingsResult{}, err
		}
		result.Total = count
	}

	return result, nil
}

func (s *FindingsStore) GetContentTypes(ctx context.Context, scanID string) ([]string, error) {
	const query = `SELECT DISTINCT content_type FROM scan_findings WHERE scan_id = ? AND content_type != '' ORDER BY content_type ASC`
	rows, err := s.db.QueryContext(ctx, query, scanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var types []string
	for rows.Next() {
		var ct string
		if err := rows.Scan(&ct); err != nil {
			return nil, err
		}
		types = append(types, ct)
	}
	return types, rows.Err()
}
