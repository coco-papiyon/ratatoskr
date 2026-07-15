package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type archiveEntry struct {
	source string
	name   string
}

var entries = []archiveEntry{
	{source: "data/README.md", name: "README.md"},
	{source: "data/notes.txt", name: "docs/notes.txt"},
	{source: "data/structured/records.json", name: "structured/records.json"},
	{source: "data/structured/sample.csv", name: "structured/sample.csv"},
	{source: "data/images/test-image.svg", name: "images/test-image.svg"},
}

var archiveTime = time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC)

func main() {
	outputDirectory := filepath.Join("data", "compress")
	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		panic(err)
	}
	zipPath := filepath.Join(outputDirectory, "sample-archive.zip")
	tarGzPath := filepath.Join(outputDirectory, "sample-archive.tar.gz")
	if err := writeZIP(zipPath); err != nil {
		panic(err)
	}
	if err := writeTarGz(tarGzPath); err != nil {
		panic(err)
	}
	fmt.Printf("generated %s and %s\n", zipPath, tarGzPath)
}

func writeZIP(outputPath string) error {
	output, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer output.Close()

	writer := zip.NewWriter(output)
	defer writer.Close()
	for _, entry := range entries {
		contents, err := os.ReadFile(filepath.Clean(entry.source))
		if err != nil {
			return err
		}
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		header.SetModTime(archiveTime)
		destination, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err := destination.Write(contents); err != nil {
			return err
		}
	}
	return nil
}

func writeTarGz(outputPath string) error {
	output, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer output.Close()

	gzipWriter := gzip.NewWriter(output)
	gzipWriter.Header.ModTime = archiveTime
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		source, err := os.Open(filepath.Clean(entry.source))
		if err != nil {
			return err
		}
		info, err := source.Stat()
		if err != nil {
			source.Close()
			return err
		}
		header := &tar.Header{Name: entry.name, Mode: 0o644, Size: info.Size(), ModTime: archiveTime}
		if err := tarWriter.WriteHeader(header); err != nil {
			source.Close()
			return err
		}
		if _, err := io.Copy(tarWriter, source); err != nil {
			source.Close()
			return err
		}
		if err := source.Close(); err != nil {
			return err
		}
	}
	if err := tarWriter.Close(); err != nil {
		return err
	}
	return gzipWriter.Close()
}
