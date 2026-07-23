import { describe, expect, it } from "vitest";
import { ansiToHtml, highlightCode, markdownToHtml, parseCsv, showMermaidError } from "./viewer-format";

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

  it("does not convert fenced code block content into HTML lists", () => {
    const html = markdownToHtml("```text\n- item\n**bold**\n```\n");
    expect(html).toContain("<pre><code>- item\n**bold**\n</code></pre>");
    expect(html).not.toContain("<ul>");
    expect(html).not.toContain("<strong>");
  });

  it("marks Mermaid blocks for diagram rendering without applying Markdown formatting", () => {
    const html = markdownToHtml("```mermaid\nflowchart LR\n  A[**source**] --> B\n```\n");
    expect(html).toContain('<div class="mermaid-diagram">flowchart LR\n  A[**source**] --&gt; B\n</div>');
    expect(html).not.toContain("<strong>source</strong>");
  });

  it("keeps PlantUML blocks as escaped source code", () => {
    const html = markdownToHtml("```plantuml\n@startuml\nAlice -> Bob: hello\n@enduml\n```\n");
    expect(html).toContain("<pre><code>@startuml\nAlice -&gt; Bob: hello\n@enduml\n</code></pre>");
    expect(html).not.toContain("mermaid-diagram");
  });

  it("renders relative and external Markdown links for click handling", () => {
    const html = markdownToHtml("[guide](docs/guide.md) [site](https://example.com/docs?a=1&b=2)");
    expect(html).toContain('<a class="markdown-link" href="docs/guide.md">guide</a>');
    expect(html).toContain('<a class="markdown-link" href="https://example.com/docs?a=1&amp;b=2">site</a>');
  });

  it("replaces failed Mermaid output with an error message and hides its source", () => {
    const classes = new Set<string>();
    const diagram = {
      classList: { add: (name: string) => classes.add(name) },
      textContent: "invalid Mermaid source",
    } as unknown as HTMLElement;
    showMermaidError(diagram);
    expect(classes).toContain("mermaid-diagram-error");
    expect(diagram.textContent).toBe("Mermaid図を描画できませんでした。");
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
