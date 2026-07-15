package app

type LocalEntry struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Path         string `json:"path"`
	ModifiedAt   string `json:"modifiedAt,omitempty"`
	Size         int64  `json:"size,omitempty"`
	ArchivePath  string `json:"archivePath,omitempty"`
	ArchiveEntry string `json:"archiveEntry,omitempty"`
}

type S3Preview struct {
	Content string `json:"content"`
	DataURL string `json:"dataUrl,omitempty"`
}

type ViewerConfig struct {
	Extensions  map[string][]string `json:"extensions"`
	Proxy       string              `json:"proxy"`
	Certificate string              `json:"certificate"`
}

type StructuredTableRule struct {
	Name        string `json:"name" yaml:"name"`
	FilePattern string `json:"filePattern" yaml:"filePattern"`
	JQ          string `json:"jq" yaml:"jq"`
}

type StructuredTable struct {
	RuleName string     `json:"ruleName"`
	Columns  []string   `json:"columns"`
	Rows     [][]string `json:"rows"`
}
