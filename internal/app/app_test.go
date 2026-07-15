package app

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestConvertStructuredToTableDefaultRule(t *testing.T) {
	app := &App{structuredTableRules: DefaultStructuredTableRules()}
	table, err := app.ConvertStructuredToTable("records.json", `[
		{"id": 1, "name": "Ratatoskr", "enabled": true},
		{"id": 2, "name": "Archive", "enabled": false}
	]`)
	if err != nil {
		t.Fatalf("ConvertStructuredToTable returned an error: %v", err)
	}
	if table == nil {
		t.Fatal("ConvertStructuredToTable returned no table")
	}
	if !reflect.DeepEqual(table.Columns, []string{"enabled", "id", "name"}) {
		t.Fatalf("unexpected columns: %#v", table.Columns)
	}
	if !reflect.DeepEqual(table.Rows, [][]string{{"true", "1", "Ratatoskr"}, {"false", "2", "Archive"}}) {
		t.Fatalf("unexpected rows: %#v", table.Rows)
	}
}

func TestConvertStructuredToTableCustomJQRule(t *testing.T) {
	app := &App{structuredTableRules: []StructuredTableRule{{
		Name:        "Enabled records",
		FilePattern: `^config\.json$`,
		JQ:          `.items[] | select(.enabled) | {identifier: .id, label: .name}`,
	}}}
	table, err := app.ConvertStructuredToTable("config.json", `{
		"items": [
			{"id": 1, "name": "shown", "enabled": true},
			{"id": 2, "name": "hidden", "enabled": false}
		]
	}`)
	if err != nil {
		t.Fatalf("ConvertStructuredToTable returned an error: %v", err)
	}
	if table == nil || table.RuleName != "Enabled records" {
		t.Fatalf("unexpected table: %#v", table)
	}
	if !reflect.DeepEqual(table.Columns, []string{"identifier", "label"}) {
		t.Fatalf("unexpected columns: %#v", table.Columns)
	}
	if !reflect.DeepEqual(table.Rows, [][]string{{"1", "shown"}}) {
		t.Fatalf("unexpected rows: %#v", table.Rows)
	}
}

func TestConvertStructuredToTableFixedArrayUsesFirstRowAsHeaders(t *testing.T) {
	app := &App{structuredTableRules: []StructuredTableRule{{
		Name:        "Fixed array",
		FilePattern: `fixed-array\.json$`,
		JQ:          ".",
	}}}
	table, err := app.ConvertStructuredToTable("fixed-array.json", `[
		["title1", "title2"],
		["val1", "val2"],
		["val3", "val4"]
	]`)
	if err != nil {
		t.Fatalf("ConvertStructuredToTable returned an error: %v", err)
	}
	if table == nil {
		t.Fatal("ConvertStructuredToTable returned no table")
	}
	if !reflect.DeepEqual(table.Columns, []string{"title1", "title2"}) {
		t.Fatalf("unexpected columns: %#v", table.Columns)
	}
	if !reflect.DeepEqual(table.Rows, [][]string{{"val1", "val2"}, {"val3", "val4"}}) {
		t.Fatalf("unexpected rows: %#v", table.Rows)
	}
}

func TestConvertStructuredToTableIgnoresUnmatchedFiles(t *testing.T) {
	app := &App{structuredTableRules: DefaultStructuredTableRules()}
	table, err := app.ConvertStructuredToTable("records.xml", `{}`)
	if err != nil {
		t.Fatalf("ConvertStructuredToTable returned an error: %v", err)
	}
	if table != nil {
		t.Fatalf("expected no table, got %#v", table)
	}
}

func TestConvertStructuredToTableUsesFirstMatchingRule(t *testing.T) {
	app := &App{structuredTableRules: []StructuredTableRule{
		{Name: "specific", FilePattern: `records\.json$`, JQ: `.[0] | {specific: .name}`},
		{Name: "fallback", FilePattern: `\.json$`, JQ: `.[0] | {fallback: .name}`},
	}}
	table, err := app.ConvertStructuredToTable("records.json", `[{"name":"first"}]`)
	if err != nil {
		t.Fatalf("ConvertStructuredToTable returned an error: %v", err)
	}
	if table == nil || table.RuleName != "specific" {
		t.Fatalf("expected the first rule, got %#v", table)
	}
}

func TestConvertStructuredToTableYAML(t *testing.T) {
	app := &App{structuredTableRules: []StructuredTableRule{{
		Name:        "YAML services",
		FilePattern: `\.ya?ml$`,
		JQ:          `.services[] | {name, port}`,
	}}}
	table, err := app.ConvertStructuredToTable("services.yaml", "services:\n  - name: api\n    port: 8080\n  - name: web\n    port: 3000\n")
	if err != nil {
		t.Fatalf("ConvertStructuredToTable returned an error: %v", err)
	}
	if table == nil {
		t.Fatal("ConvertStructuredToTable returned no table")
	}
	if !reflect.DeepEqual(table.Columns, []string{"name", "port"}) {
		t.Fatalf("unexpected columns: %#v", table.Columns)
	}
	if !reflect.DeepEqual(table.Rows, [][]string{{"api", "8080"}, {"web", "3000"}}) {
		t.Fatalf("unexpected rows: %#v", table.Rows)
	}
}

func TestValidateStructuredTableRulesRejectsInvalidPatternAndJQ(t *testing.T) {
	tests := []StructuredTableRule{
		{Name: "invalid pattern", FilePattern: "[", JQ: "."},
		{Name: "invalid jq", FilePattern: `\.json$`, JQ: "if"},
	}
	for _, rule := range tests {
		if err := validateStructuredTableRules([]StructuredTableRule{rule}); err == nil {
			t.Fatalf("expected validation error for %s", rule.Name)
		}
	}
}

func TestZipArchiveNavigationAndPreview(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, file := range []struct{ name, content string }{
		{"root.txt", "root content"},
		{"docs/readme.md", "# Archive README"},
		{"docs/nested/config.json", `{"enabled":true}`},
	} {
		entry, err := writer.Create(file.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(file.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	root, err := listArchiveEntries("sample.zip", "", buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(root) != 2 || root[0].Name != "docs" || root[0].Kind != "folder" || root[1].Name != "root.txt" {
		t.Fatalf("unexpected root entries: %#v", root)
	}
	docs, err := listArchiveEntries("sample.zip", "docs/", buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 || docs[0].Name != "nested" || docs[1].Name != "readme.md" {
		t.Fatalf("unexpected docs entries: %#v", docs)
	}
	preview, err := readArchivePreview("sample.zip", "docs/readme.md", "utf-8", buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if preview.Content != "# Archive README" {
		t.Fatalf("unexpected preview: %#v", preview)
	}
}

func TestTarGzArchiveNavigationAndPreview(t *testing.T) {
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	contents := []byte("name: ratatoskr\nenabled: true\n")
	if err := tarWriter.WriteHeader(&tar.Header{Name: "config/app.yaml", Mode: 0o644, Size: int64(len(contents))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write(contents); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}

	root, err := listArchiveEntries("sample.tar.gz", "", buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(root) != 1 || root[0].Name != "config" || root[0].Kind != "folder" {
		t.Fatalf("unexpected root entries: %#v", root)
	}
	preview, err := readArchivePreview("sample.tar.gz", "config/app.yaml", "utf-8", buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if preview.Content != string(contents) {
		t.Fatalf("unexpected preview: %#v", preview)
	}
}

func TestGeneratedArchivesContainPreviewableImage(t *testing.T) {
	for _, archivePath := range []string{filepath.Join("..", "..", "data", "compress", "sample-archive.zip"), filepath.Join("..", "..", "data", "compress", "sample-archive.tar.gz")} {
		contents, err := os.ReadFile(archivePath)
		if err != nil {
			t.Fatalf("read %s: %v", archivePath, err)
		}
		entries, err := listArchiveEntries(archivePath, "images/", contents)
		if err != nil {
			t.Fatalf("list %s: %v", archivePath, err)
		}
		if len(entries) != 1 || entries[0].Name != "test-image.svg" {
			t.Fatalf("unexpected image entries in %s: %#v", archivePath, entries)
		}
		preview, err := readArchivePreview(archivePath, "images/test-image.svg", "utf-8", contents)
		if err != nil {
			t.Fatalf("preview %s: %v", archivePath, err)
		}
		if !strings.HasPrefix(preview.DataURL, "data:image/svg+xml;base64,") {
			t.Fatalf("image in %s was not converted to a data URL", archivePath)
		}
	}
}
