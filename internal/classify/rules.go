package classify

// Category constants.
const (
	CatDocument   = "document"
	CatArchive    = "archive"
	CatMedia      = "media"
	CatSoftware   = "software"
	CatDatabase   = "database"
	CatSourceCode = "source_code"
	CatConfig     = "config"
	CatOther      = "other"
)

// extCategory maps lowercase file extensions to their primary category.
var extCategory = map[string]string{
	// Documents
	".pdf":  CatDocument,
	".doc":  CatDocument,
	".docx": CatDocument,
	".xls":  CatDocument,
	".xlsx": CatDocument,
	".ppt":  CatDocument,
	".pptx": CatDocument,
	".odt":  CatDocument,
	".ods":  CatDocument,
	".odp":  CatDocument,
	".rtf":  CatDocument,
	".tex":  CatDocument,
	".epub": CatDocument,
	".csv":  CatDocument,
	".tsv":  CatDocument,
	".txt":  CatDocument,
	".md":   CatDocument,
	".rst":  CatDocument,

	// Archives
	".zip":    CatArchive,
	".tar":    CatArchive,
	".gz":     CatArchive,
	".tgz":    CatArchive,
	".bz2":    CatArchive,
	".xz":     CatArchive,
	".zst":    CatArchive,
	".7z":     CatArchive,
	".rar":    CatArchive,
	".cab":    CatArchive,
	".wpress": CatArchive,

	// Media
	".mp3":  CatMedia,
	".mp4":  CatMedia,
	".avi":  CatMedia,
	".mkv":  CatMedia,
	".mov":  CatMedia,
	".flac": CatMedia,
	".wav":  CatMedia,
	".ogg":  CatMedia,
	".webm": CatMedia,
	".wmv":  CatMedia,
	".flv":  CatMedia,
	".m4a":  CatMedia,
	".m4v":  CatMedia,
	".aac":  CatMedia,
	".svg":  CatMedia,
	".png":  CatMedia,
	".jpg":  CatMedia,
	".jpeg": CatMedia,
	".gif":  CatMedia,
	".bmp":  CatMedia,
	".webp": CatMedia,
	".ico":  CatMedia,
	".tiff": CatMedia,

	// Software
	".exe":     CatSoftware,
	".msi":     CatSoftware,
	".deb":     CatSoftware,
	".rpm":     CatSoftware,
	".apk":     CatSoftware,
	".dmg":     CatSoftware,
	".iso":     CatSoftware,
	".img":     CatSoftware,
	".appimage": CatSoftware,
	".flatpak": CatSoftware,
	".snap":    CatSoftware,

	// Database
	".sql":    CatDatabase,
	".db":     CatDatabase,
	".sqlite": CatDatabase,
	".sqlite3": CatDatabase,
	".mdb":    CatDatabase,
	".accdb":  CatDatabase,
	".dump":   CatDatabase,
	".bson":   CatDatabase,

	// Source code
	".go":    CatSourceCode,
	".py":    CatSourceCode,
	".js":    CatSourceCode,
	".ts":    CatSourceCode,
	".jsx":   CatSourceCode,
	".tsx":   CatSourceCode,
	".java":  CatSourceCode,
	".c":     CatSourceCode,
	".cpp":   CatSourceCode,
	".h":     CatSourceCode,
	".hpp":   CatSourceCode,
	".rs":    CatSourceCode,
	".rb":    CatSourceCode,
	".php":   CatSourceCode,
	".swift": CatSourceCode,
	".kt":    CatSourceCode,
	".scala": CatSourceCode,
	".cs":    CatSourceCode,
	".lua":   CatSourceCode,
	".pl":    CatSourceCode,
	".sh":    CatSourceCode,
	".bash":  CatSourceCode,
	".zsh":   CatSourceCode,
	".ps1":   CatSourceCode,
	".bat":   CatSourceCode,
	".r":     CatSourceCode,
	".m":     CatSourceCode,
	".vue":   CatSourceCode,
	".elm":   CatSourceCode,

	// Config
	".env":        CatConfig,
	".ini":        CatConfig,
	".conf":       CatConfig,
	".cfg":        CatConfig,
	".yaml":       CatConfig,
	".yml":        CatConfig,
	".toml":       CatConfig,
	".json":       CatConfig,
	".xml":        CatConfig,
	".properties": CatConfig,
	".htaccess":   CatConfig,
	".htpasswd":   CatConfig,
	".nginxconf":  CatConfig,
}

// mimeCategory maps MIME type prefixes to categories.
// Checked in order; first match wins.
var mimeCategory = []struct {
	prefix   string
	category string
}{
	{"application/pdf", CatDocument},
	{"application/msword", CatDocument},
	{"application/vnd.openxmlformats-officedocument", CatDocument},
	{"application/vnd.oasis.opendocument", CatDocument},
	{"text/csv", CatDocument},
	{"text/plain", CatDocument},
	{"text/markdown", CatDocument},
	{"application/zip", CatArchive},
	{"application/gzip", CatArchive},
	{"application/x-tar", CatArchive},
	{"application/x-7z-compressed", CatArchive},
	{"application/x-rar", CatArchive},
	{"application/x-bzip2", CatArchive},
	{"application/x-xz", CatArchive},
	{"application/zstd", CatArchive},
	{"audio/", CatMedia},
	{"video/", CatMedia},
	{"image/", CatMedia},
	{"application/x-executable", CatSoftware},
	{"application/x-msdos-program", CatSoftware},
	{"application/x-deb", CatSoftware},
	{"application/x-rpm", CatSoftware},
	{"application/x-iso9660-image", CatSoftware},
	{"application/sql", CatDatabase},
	{"application/x-sqlite3", CatDatabase},
	{"text/x-python", CatSourceCode},
	{"text/x-go", CatSourceCode},
	{"text/x-java", CatSourceCode},
	{"text/x-c", CatSourceCode},
	{"text/x-sh", CatSourceCode},
	{"text/x-script", CatSourceCode},
	{"text/javascript", CatSourceCode},
	{"application/javascript", CatSourceCode},
	{"application/json", CatConfig},
	{"application/xml", CatConfig},
	{"text/xml", CatConfig},
	{"text/yaml", CatConfig},
}

// sensitiveExts are file extensions that indicate potentially sensitive files.
var sensitiveExts = map[string]bool{
	".env":      true,
	".key":      true,
	".pem":      true,
	".p12":      true,
	".pfx":      true,
	".jks":      true,
	".keystore": true,
	".htpasswd": true,
	".shadow":   true,
	".passwd":   true,
	".pgpass":   true,
	".netrc":    true,
	".npmrc":    true,
	".pypirc":   true,
}

// sensitivePatterns are substrings in filenames that suggest sensitive content.
var sensitivePatterns = []string{
	"password",
	"passwd",
	"secret",
	"credential",
	"private_key",
	"id_rsa",
	"id_ed25519",
}

// backupPatterns are substrings in filenames that suggest backup files.
var backupPatterns = []string{
	"backup",
	"bak",
	"old",
	"copy",
	"snapshot",
}

// rareExts get a bonus interest score — files that are uncommon and worth investigating.
var rareExts = map[string]bool{
	".sql":      true,
	".dump":     true,
	".env":      true,
	".key":      true,
	".pem":      true,
	".bak":      true,
	".conf":     true,
	".htpasswd": true,
	".shadow":   true,
	".pgpass":   true,
	".sqlite":   true,
	".sqlite3":  true,
	".mdb":      true,
	".wpress":   true,
}

// baseCategoryScore assigns a base interest score per category.
var baseCategoryScore = map[string]int{
	CatDatabase:   40,
	CatConfig:     30,
	CatSourceCode: 25,
	CatSoftware:   20,
	CatArchive:    15,
	CatDocument:   10,
	CatMedia:      5,
	CatOther:      5,
}
