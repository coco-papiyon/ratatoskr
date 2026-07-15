import { describe, expect, it } from "vitest";
import { ansiToHtml, highlightCode, markdownToHtml, parseCsv } from "./viewer-format";

describe("parseCsv", () => {
  it("parses quoted commas, escaped quotes, and line breaks", () => {
    expect(parseCsv('name,note\r\nRatatoskr,"one,two"\r\nViewer,"say ""hello"""')).toEqual([
      ["name", "note"],
      ["Ratatoskr", "one,two"],
      ["Viewer", 'say "hello"'],
    ]);
  });

  it("keeps a line break inside a quoted cell", () => {
    expect(parseCsv('id,message\n1,"first\nsecond"')).toEqual([["id", "message"], ["1", "first\nsecond"]]);
  });
});

describe("viewer HTML formatting", () => {
  it("renders representative Markdown structures", () => {
    const html = markdownToHtml("| Name | Value |\n| --- | --- |\n| a | 1 |\n\n> quoted\n\n1. first\n2. second\n");
    expect(html).toContain("<table>");
    expect(html).toContain("<blockquote>quoted</blockquote>");
    expect(html).toContain("<ol><li>first</li><li>second</li></ol>");
  });

  it("escapes source code and applies token classes", () => {
    const html = highlightCode('const value = "<tag>";');
    expect(html).toContain('<span class="code-keyword">const</span>');
    expect(html).toContain("&lt;tag&gt;");
  });

  it("converts ANSI colors without exposing escape codes", () => {
    const html = ansiToHtml("normal \u001b[31merror\u001b[0m done");
    expect(html).toContain('<span class="ansi-color-31">error</span>');
    expect(html).not.toContain("\u001b[");
  });
});
