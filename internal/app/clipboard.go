package app

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

var clipboardMutex sync.Mutex

type ClipboardConversion struct {
	Input       string `json:"input"`
	Output      string `json:"output"`
	InputFormat string `json:"inputFormat"`
}

const (
	clipboardUnicodeText uint = 13
	clipboardHTML        uint = 49362
	clipboardRTF         uint = 49303
)

func (a *App) ConvertClipboardTableToMarkdown() (*ClipboardConversion, error) {
	clipboardMutex.Lock()
	defer clipboardMutex.Unlock()
	input, format, err := readSystemClipboard()
	if err != nil {
		return nil, err
	}
	output, err := clipboardToMarkdown(input, format)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(output) == "" {
		return nil, errors.New("変換結果が空です")
	}
	if err := writeSystemClipboard(output); err != nil {
		return nil, fmt.Errorf("クリップボードへの書き込みに失敗しました: %w", err)
	}
	return &ClipboardConversion{Input: input, Output: output, InputFormat: clipboardFormatName(format)}, nil
}

func (a *App) ConvertClipboardMarkdownToTable() (*ClipboardConversion, error) {
	clipboardMutex.Lock()
	defer clipboardMutex.Unlock()
	input, _, err := readSystemClipboard()
	if err != nil {
		return nil, err
	}
	output, err := markdownTableToTSV(input)
	if err != nil {
		return nil, err
	}
	if err := writeSystemClipboard(output); err != nil {
		return nil, fmt.Errorf("クリップボードへの書き込みに失敗しました: %w", err)
	}
	return &ClipboardConversion{Input: input, Output: output, InputFormat: "MARKDOWN"}, nil
}

func (a *App) CopyMarkdownTableToClipboard(markdown string) error {
	_, err := a.ConvertMarkdownTableToTSV(markdown)
	return err
}

func (a *App) ConvertMarkdownTableToTSV(markdown string) (string, error) {
	clipboardMutex.Lock()
	defer clipboardMutex.Unlock()
	output, err := markdownTableToTSV(markdown)
	if err != nil {
		return "", err
	}
	if err := writeSystemClipboard(output); err != nil {
		return "", fmt.Errorf("クリップボードへの書き込みに失敗しました: %w", err)
	}
	return output, nil
}

func clipboardToMarkdown(input string, format uint) (string, error) {
	switch format {
	case clipboardUnicodeText:
		if isMarkdownTable(input) {
			return "", errors.New("クリップボードは既にMarkdown形式です")
		}
		if !strings.Contains(strings.ReplaceAll(input, "\r\n", "\n"), "\t") {
			return "", errors.New("クリップボードに表形式のデータがありません")
		}
		return textTableToMarkdown(input), nil
	case clipboardHTML:
		return htmlTableToMarkdown(input)
	case clipboardRTF:
		plainText := rtfToPlainText(input)
		if isMarkdownTable(plainText) {
			return "", errors.New("クリップボードは既にMarkdown形式です")
		}
		if !strings.Contains(plainText, "\t") {
			return "", errors.New("クリップボードに表形式のデータがありません")
		}
		return textTableToMarkdown(plainText), nil
	default:
		return "", fmt.Errorf("未対応のクリップボード形式です: %d", format)
	}
}

func clipboardFormatName(format uint) string {
	switch format {
	case clipboardUnicodeText:
		return "TEXT"
	case clipboardHTML:
		return "HTML"
	case clipboardRTF:
		return "RTF"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", format)
	}
}

func textTableToMarkdown(input string) string {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	lines := strings.Split(input, "\n")
	rows := make([][]string, 0)
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		rows = append(rows, splitClipboardRow(line))
	}
	return tableToMarkdown(rows)
}

func splitClipboardRow(line string) []string {
	if strings.Contains(line, "\t") {
		cells := strings.Split(line, "\t")
		for index := range cells {
			cells[index] = strings.TrimSpace(cells[index])
		}
		return cells
	}
	return strings.Fields(line)
}

func htmlTableToMarkdown(input string) (string, error) {
	document, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return "", fmt.Errorf("HTMLを解析できません: %w", err)
	}
	table := findHTMLTable(document)
	if len(table) == 0 {
		return "", errors.New("クリップボード内に表が見つかりません")
	}
	return tableToMarkdown(table), nil
}

func findHTMLTable(node *html.Node) [][]string {
	if node.Type == html.ElementNode && node.Data == "table" {
		return extractHTMLTable(node)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if table := findHTMLTable(child); len(table) > 0 {
			return table
		}
	}
	return nil
}

func extractHTMLTable(tableNode *html.Node) [][]string {
	rows := make([][]string, 0)
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "tr" {
			row := make([]string, 0)
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.ElementNode && (child.Data == "th" || child.Data == "td") {
					row = append(row, htmlNodeText(child))
				}
			}
			if len(row) > 0 {
				rows = append(rows, row)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(tableNode)
	return rows
}

func htmlNodeText(node *html.Node) string {
	if node.Type == html.TextNode {
		return strings.TrimSpace(node.Data)
	}
	var builder strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		builder.WriteString(htmlNodeText(child))
	}
	return strings.TrimSpace(builder.String())
}

func rtfToPlainText(input string) string {
	var builder strings.Builder
	ignoreDepth := 0
	for index := 0; index < len(input); {
		switch input[index] {
		case '{':
			if ignoreDepth > 0 {
				ignoreDepth++
			}
			index++
		case '}':
			if ignoreDepth > 0 {
				ignoreDepth--
			}
			index++
		case '\\':
			index++
			if index >= len(input) {
				break
			}
			if input[index] == '\\' || input[index] == '{' || input[index] == '}' {
				if ignoreDepth == 0 {
					builder.WriteByte(input[index])
				}
				index++
				continue
			}
			start := index
			for index < len(input) && ((input[index] >= 'a' && input[index] <= 'z') || (input[index] >= 'A' && input[index] <= 'Z')) {
				index++
			}
			word := input[start:index]
			if word == "par" || word == "line" || word == "row" {
				builder.WriteByte('\n')
			} else if word == "tab" || word == "cell" {
				builder.WriteByte('\t')
			}
			for index < len(input) && input[index] == ' ' {
				index++
			}
		default:
			if ignoreDepth == 0 && input[index] != '\r' && input[index] != '\n' {
				builder.WriteByte(input[index])
			}
			index++
		}
	}
	return builder.String()
}

func tableToMarkdown(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	width := 0
	for _, row := range rows {
		if len(row) > width {
			width = len(row)
		}
	}
	var builder strings.Builder
	for rowIndex, row := range rows {
		builder.WriteString("| ")
		for column := 0; column < width; column++ {
			if column > 0 {
				builder.WriteString(" | ")
			}
			if column < len(row) {
				builder.WriteString(strings.ReplaceAll(strings.TrimSpace(row[column]), "|", "\\|"))
			}
		}
		builder.WriteString(" |\n")
		if rowIndex == 0 {
			builder.WriteString("| ")
			builder.WriteString(strings.TrimRight(strings.Repeat("--- | ", width), " |"))
			builder.WriteString(" |\n")
		}
	}
	return builder.String()
}

func markdownTableToTSV(markdown string) (string, error) {
	rows := make([][]string, 0)
	for _, line := range strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "|") {
			continue
		}
		cells := splitMarkdownRow(line)
		if len(cells) == 0 || isMarkdownSeparator(cells) {
			continue
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 {
		return "", errors.New("Markdown内に表が見つかりません")
	}
	width := 0
	for _, row := range rows {
		if len(row) > width {
			width = len(row)
		}
	}
	var builder strings.Builder
	for rowIndex, row := range rows {
		if rowIndex > 0 {
			builder.WriteByte('\n')
		}
		for column := 0; column < width; column++ {
			if column > 0 {
				builder.WriteByte('\t')
			}
			if column < len(row) {
				builder.WriteString(strings.ReplaceAll(row[column], "\t", " "))
			}
		}
	}
	return builder.String(), nil
}

func isMarkdownTable(markdown string) bool {
	dataRows := 0
	separatorFound := false
	for _, line := range strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "|") {
			continue
		}
		cells := splitMarkdownRow(line)
		if len(cells) == 0 {
			continue
		}
		if isMarkdownSeparator(cells) {
			separatorFound = true
			continue
		}
		dataRows++
	}
	return separatorFound && dataRows > 0
}

func splitMarkdownRow(line string) []string {
	var parts []string
	var cell strings.Builder
	line = strings.TrimSpace(line)
	for index := 0; index < len(line); index++ {
		if line[index] == '\\' && index+1 < len(line) && line[index+1] == '|' {
			cell.WriteByte('|')
			index++
			continue
		}
		if line[index] == '|' {
			parts = append(parts, strings.TrimSpace(cell.String()))
			cell.Reset()
			continue
		}
		cell.WriteByte(line[index])
	}
	parts = append(parts, strings.TrimSpace(cell.String()))
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func isMarkdownSeparator(row []string) bool {
	if len(row) == 0 {
		return false
	}
	for _, cell := range row {
		cell = strings.TrimSpace(cell)
		if len(cell) < 3 || strings.Trim(cell, "-:") != "" {
			return false
		}
	}
	return true
}
