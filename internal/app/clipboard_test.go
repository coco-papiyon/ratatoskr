package app

import "testing"

func TestTextTableToMarkdown(t *testing.T) {
	actual := textTableToMarkdown("Name\tCount\nAlpha\t1\nBeta\t2")
	expected := "| Name | Count |\n| --- | --- |\n| Alpha | 1 |\n| Beta | 2 |\n"
	if actual != expected {
		t.Fatalf("unexpected Markdown:\n%s", actual)
	}
}

func TestHTMLTableToMarkdown(t *testing.T) {
	actual, err := htmlTableToMarkdown(`<html><body><table><tr><th>Name</th><th>Value</th></tr><tr><td>A</td><td>1</td></tr></table></body></html>`)
	if err != nil {
		t.Fatalf("htmlTableToMarkdown returned an error: %v", err)
	}
	expected := "| Name | Value |\n| --- | --- |\n| A | 1 |\n"
	if actual != expected {
		t.Fatalf("unexpected Markdown:\n%s", actual)
	}
}

func TestClipboardToMarkdownRejectsMarkdownTable(t *testing.T) {
	_, err := clipboardToMarkdown("| Name | Value |\n| --- | --- |\n| A | 1 |", clipboardUnicodeText)
	if err == nil {
		t.Fatal("expected Markdown input to be rejected")
	}
}

func TestClipboardToMarkdownRejectsPlainText(t *testing.T) {
	_, err := clipboardToMarkdown("ただのテキストです", clipboardUnicodeText)
	if err == nil {
		t.Fatal("expected plain text input to be rejected")
	}
}

func TestClipboardToMarkdownAcceptsTabSeparatedTable(t *testing.T) {
	actual, err := clipboardToMarkdown("Name\tValue\nA\t1", clipboardUnicodeText)
	if err != nil {
		t.Fatalf("clipboardToMarkdown returned an error: %v", err)
	}
	if actual != "| Name | Value |\n| --- | --- |\n| A | 1 |\n" {
		t.Fatalf("unexpected Markdown: %q", actual)
	}
}

func TestMarkdownTableToTSV(t *testing.T) {
	actual, err := markdownTableToTSV("| Name | Note |\n| --- | --- |\n| Alpha | A \\| B |\n| Beta | Two\n")
	if err != nil {
		t.Fatalf("markdownTableToTSV returned an error: %v", err)
	}
	if actual != "Name\tNote\nAlpha\tA | B\nBeta\tTwo" {
		t.Fatalf("unexpected TSV: %q", actual)
	}
}
