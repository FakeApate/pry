package classify

import (
	"testing"
)

func TestClassifyCategory(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		contentType string
		wantCat     string
	}{
		// Extension-based
		{"pdf", "https://example.com/report.pdf", "", CatDocument},
		{"docx", "https://example.com/file.docx", "", CatDocument},
		{"csv", "https://example.com/data.csv", "", CatDocument},
		{"txt", "https://example.com/readme.txt", "", CatDocument},
		{"md", "https://example.com/README.md", "", CatDocument},
		{"zip", "https://example.com/archive.zip", "", CatArchive},
		{"tar.gz uses .gz", "https://example.com/backup.tar.gz", "", CatArchive},
		{"7z", "https://example.com/data.7z", "", CatArchive},
		{"wpress", "https://example.com/site.wpress", "", CatArchive},
		{"mp4", "https://example.com/video.mp4", "", CatMedia},
		{"jpg", "https://example.com/photo.jpg", "", CatMedia},
		{"png", "https://example.com/icon.png", "", CatMedia},
		{"exe", "https://example.com/setup.exe", "", CatSoftware},
		{"iso", "https://example.com/os.iso", "", CatSoftware},
		{"deb", "https://example.com/pkg.deb", "", CatSoftware},
		{"sql", "https://example.com/dump.sql", "", CatDatabase},
		{"sqlite", "https://example.com/app.sqlite", "", CatDatabase},
		{"db", "https://example.com/data.db", "", CatDatabase},
		{"go", "https://example.com/main.go", "", CatSourceCode},
		{"py", "https://example.com/script.py", "", CatSourceCode},
		{"js", "https://example.com/app.js", "", CatSourceCode},
		{"php", "https://example.com/index.php", "", CatSourceCode},
		{"sh", "https://example.com/deploy.sh", "", CatSourceCode},
		{"env", "https://example.com/.env", "", CatConfig},
		{"yaml", "https://example.com/config.yaml", "", CatConfig},
		{"json", "https://example.com/package.json", "", CatConfig},
		{"xml", "https://example.com/web.xml", "", CatConfig},
		{"toml", "https://example.com/pyproject.toml", "", CatConfig},
		{"htpasswd", "https://example.com/.htpasswd", "", CatConfig},

		// MIME-based fallback (no recognizable extension)
		{"mime pdf", "https://example.com/download", "application/pdf", CatDocument},
		{"mime zip", "https://example.com/file", "application/zip", CatArchive},
		{"mime gzip", "https://example.com/file", "application/gzip", CatArchive},
		{"mime audio", "https://example.com/track", "audio/mpeg", CatMedia},
		{"mime video", "https://example.com/clip", "video/mp4", CatMedia},
		{"mime image", "https://example.com/pic", "image/jpeg", CatMedia},
		{"mime javascript", "https://example.com/bundle", "text/javascript", CatSourceCode},
		{"mime json", "https://example.com/api", "application/json", CatConfig},
		{"mime python", "https://example.com/script", "text/x-python", CatSourceCode},
		{"mime sql", "https://example.com/data", "application/sql", CatDatabase},
		{"mime xml", "https://example.com/feed", "application/xml", CatConfig},

		// Extension takes priority over MIME
		{"ext overrides mime", "https://example.com/data.sql", "application/octet-stream", CatDatabase},

		// Unknown
		{"unknown ext and mime", "https://example.com/file.xyz", "application/octet-stream", CatOther},
		{"no ext no mime", "https://example.com/something", "", CatOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Classify(tt.url, tt.contentType, 0)
			if r.Category != tt.wantCat {
				t.Errorf("Classify(%q, %q).Category = %q, want %q", tt.url, tt.contentType, r.Category, tt.wantCat)
			}
		})
	}
}

func TestClassifyTags(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantTags []string
	}{
		{"env is sensitive", "https://example.com/.env", []string{"sensitive"}},
		{"pem is sensitive", "https://example.com/server.pem", []string{"sensitive"}},
		{"key is sensitive", "https://example.com/private.key", []string{"sensitive"}},
		{"htpasswd is sensitive", "https://example.com/.htpasswd", []string{"sensitive"}},
		{"password in name", "https://example.com/passwords.txt", []string{"sensitive"}},
		{"secret in name", "https://example.com/secret-config.json", []string{"sensitive"}},
		{"credential in name", "https://example.com/credentials.json", []string{"sensitive"}},
		{"id_rsa", "https://example.com/id_rsa", []string{"sensitive"}},
		{"backup in name", "https://example.com/backup-2026.zip", []string{"backup"}},
		{"bak extension", "https://example.com/data.bak", []string{"backup"}},
		{"old in name", "https://example.com/old-config.yaml", []string{"backup"}},
		{"log file", "https://example.com/error.log", []string{"log"}},
		{"error_log", "https://example.com/error_log", []string{"log"}},
		{"access_log", "https://example.com/access_log", []string{"log"}},
		{"backup + sensitive", "https://example.com/backup-passwords.sql", []string{"sensitive", "backup"}},
		{"no tags", "https://example.com/report.pdf", nil},
		{"normal js", "https://example.com/app.js", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Classify(tt.url, "", 0)
			if len(r.Tags) != len(tt.wantTags) {
				t.Errorf("Classify(%q).Tags = %v, want %v", tt.url, r.Tags, tt.wantTags)
				return
			}
			for i, tag := range tt.wantTags {
				if r.Tags[i] != tag {
					t.Errorf("Classify(%q).Tags[%d] = %q, want %q", tt.url, i, r.Tags[i], tag)
				}
			}
		})
	}
}

func TestClassifyInterestScore(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		mime      string
		size      int64
		wantMin   int
		wantMax   int
	}{
		{"sql file", "https://example.com/dump.sql", "", 0, 55, 70},                         // database(40) + rare(15)
		{"env file", "https://example.com/.env", "", 0, 85, 100},                             // config(30) + sensitive(40) + rare(15)
		{"pem file", "https://example.com/server.pem", "", 0, 55, 60},                        // other(5) + sensitive(40) + rare(15)
		{"large archive", "https://example.com/data.zip", "", 200_000_000, 25, 30},           // archive(15) + >100MB(10)
		{"huge sql", "https://example.com/db.sql", "", 2_000_000_000, 70, 100},               // database(40) + rare(15) + >1GB(15)
		{"password file", "https://example.com/passwords.txt", "", 0, 50, 55},                // document(10) + sensitive(40)
		{"backup sql", "https://example.com/backup.sql", "", 0, 75, 100},                     // database(40) + backup(20) + rare(15)
		{"normal pdf", "https://example.com/report.pdf", "", 1000, 10, 15},                   // document(10)
		{"normal jpg", "https://example.com/photo.jpg", "", 500_000, 5, 10},                  // media(5)
		{"unknown octet-stream", "https://example.com/file.xyz", "application/octet-stream", 0, 5, 10}, // other(5)
		{"score capped", "https://example.com/backup-password.env", "", 2_000_000_000, 100, 100},       // would exceed 100
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Classify(tt.url, tt.mime, tt.size)
			if r.InterestScore < tt.wantMin || r.InterestScore > tt.wantMax {
				t.Errorf("Classify(%q, size=%d).InterestScore = %d, want [%d, %d]",
					tt.url, tt.size, r.InterestScore, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestClassifyScoreCapped(t *testing.T) {
	r := Classify("https://example.com/backup-password.env", "", 2_000_000_000)
	if r.InterestScore > 100 {
		t.Errorf("score should be capped at 100, got %d", r.InterestScore)
	}
}

func TestClassifyURLEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"query params", "https://example.com/file.sql?v=1", CatDatabase},
		{"fragment", "https://example.com/dump.sql#top", CatDatabase},
		{"encoded path", "https://example.com/my%20file.pdf", CatDocument},
		{"trailing slash", "https://example.com/dir/", CatOther},
		{"no path", "https://example.com", CatOther},
		{"empty string", "", CatOther},
		{"path with dots", "https://example.com/v1.2.3/app.tar.gz", CatArchive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Classify(tt.url, "", 0)
			if r.Category != tt.want {
				t.Errorf("Classify(%q).Category = %q, want %q", tt.url, r.Category, tt.want)
			}
		})
	}
}

func TestExtractExt(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/file.PDF", ".pdf"},
		{"https://example.com/file.Tar.GZ", ".gz"},
		{"https://example.com/README", ""},
		{"https://example.com/dir/", ""},
		{"https://example.com/.env", ".env"},
		{"https://example.com/.htpasswd", ".htpasswd"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := extractExt(tt.url)
			if got != tt.want {
				t.Errorf("extractExt(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestExtractFilename(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/path/to/file.txt", "file.txt"},
		{"https://example.com/.env", ".env"},
		{"https://example.com/dir/", "dir"},
		{"https://example.com", "."},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := extractFilename(tt.url)
			if got != tt.want {
				t.Errorf("extractFilename(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
