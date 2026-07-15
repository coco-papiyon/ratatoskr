export function highlightCode(source: string) {
  const escaped = escapeHtml(source);
  const tokenPattern = /(\/\/.*|#.*|\/\*[\s\S]*?\*\/|"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|`(?:\\.|[^`\\])*`|\b\d+(?:\.\d+)?\b|\b(?:as|async|await|break|case|catch|class|const|continue|def|else|export|extends|false|for|from|func|function|go|if|import|in|interface|let|new|null|package|private|public|range|return|select|static|struct|switch|this|throw|true|type|var|while|with)\b)/g;
  return escaped.replace(tokenPattern, (token) => {
    if (/^(\/\/|#|\/\*)/.test(token)) return `<span class="code-comment">${token}</span>`;
    if (/^["'`]/.test(token)) return `<span class="code-string">${token}</span>`;
    if (/^\d/.test(token)) return `<span class="code-number">${token}</span>`;
    return `<span class="code-keyword">${token}</span>`;
  });
}

export function ansiToHtml(source: string) {
  let classes: string[] = [];
  let html = "";
  let cursor = 0;
  const ansiPattern = /\x1b\[([0-9;]*)m/g;
  let match: RegExpExecArray | null;
  while ((match = ansiPattern.exec(source)) !== null) {
    const segment = escapeHtml(source.slice(cursor, match.index));
    html += classes.length ? `<span class="${classes.join(" ")}">${segment}</span>` : segment;
    const codes = (match[1] || "0").split(";").map(Number);
    for (const code of codes) {
      if (code === 0) classes = [];
      else if (code === 1) classes = [...new Set([...classes, "ansi-bold"])];
      else if (code === 39) classes = classes.filter((name) => !name.startsWith("ansi-"));
      else if ((code >= 30 && code <= 37) || (code >= 90 && code <= 97)) {
        classes = classes.filter((name) => !name.startsWith("ansi-color-"));
        classes.push(`ansi-color-${code}`);
      }
    }
    cursor = match.index + match[0].length;
  }
  const tail = escapeHtml(source.slice(cursor));
  html += classes.length ? `<span class="${classes.join(" ")}">${tail}</span>` : tail;
  return html;
}

export function parseCsv(source: string) {
  const rows: string[][] = [];
  let row: string[] = [];
  let cell = "";
  let quoted = false;
  for (let index = 0; index < source.length; index += 1) {
    const character = source[index];
    const next = source[index + 1];
    if (character === '"' && quoted && next === '"') {
      cell += '"';
      index += 1;
    } else if (character === '"') {
      quoted = !quoted;
    } else if (character === "," && !quoted) {
      row.push(cell);
      cell = "";
    } else if ((character === "\n" || character === "\r") && !quoted) {
      if (character === "\r" && next === "\n") index += 1;
      row.push(cell);
      if (row.some((value) => value.length > 0)) rows.push(row);
      row = [];
      cell = "";
    } else {
      cell += character;
    }
  }
  if (cell.length > 0 || row.length > 0) {
    row.push(cell);
    rows.push(row);
  }
  return rows;
}

export function markdownToHtml(markdown: string) {
  return escapeHtml(markdown)
    .replace(/((?:^[^\S\r\n]*\|.*\|[^\S\r\n]*\r?\n?)+)/gm, (tableBlock) => markdownTableToHtml(tableBlock))
    .replace(/^### (.*)$/gm, "<h3>$1</h3>")
    .replace(/^## (.*)$/gm, "<h2>$1</h2>")
    .replace(/^# (.*)$/gm, "<h1>$1</h1>")
    .replace(/^&gt; ?([^\r\n]*(?:\r?\n&gt; ?[^\r\n]*)*)/gm, (_, quote: string) => `<blockquote>${quote.replace(/\r?\n&gt; ?/g, "<br>")}</blockquote>`)
    .replace(/(?:^\d+\. .*(?:\r?\n|$))+/gm, (listBlock) => markdownOrderedListToHtml(listBlock))
    .replace(/(?:^[-*] .*(?:\r?\n|$))+/gm, (listBlock) => markdownUnorderedListToHtml(listBlock))
    .replace(/```[\w-]*\n([\s\S]*?)```/g, "<pre><code>$1</code></pre>")
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
    .replace(/!\[([^\]]*)\]\(([^)]+)\)/g, (_, alt: string, url: string) => `<img class="markdown-image" src="${url.replace(/"/g, "&quot;")}" alt="${alt.replace(/"/g, "&quot;")}" />`)
    .replace(/\[([^\]]+)\]\((https?:\/\/[^)]+)\)/g, '<a href="$2" target="_blank" rel="noreferrer">$1</a>')
    .replace(/\n\n/g, "</p><p>")
    .replace(/^(?!<[hublp])/gm, "")
    .replace(/<p><\/p>/g, "");
}

function escapeHtml(value: string) {
  return value.replace(/[&<>]/g, (character) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" })[character] ?? character);
}

function markdownOrderedListToHtml(listBlock: string) {
  const items = listBlock.trim().split(/\r?\n/).map((line) => line.replace(/^\d+\.\s+/, ""));
  return `<ol>${items.map((item) => `<li>${item}</li>`).join("")}</ol>`;
}

function markdownUnorderedListToHtml(listBlock: string) {
  const items = listBlock.trim().split(/\r?\n/).map((line) => line.replace(/^[-*]\s+/, ""));
  return `<ul>${items.map((item) => `<li>${item}</li>`).join("")}</ul>`;
}

function markdownTableToHtml(tableBlock: string) {
  const rows = tableBlock.trim().split(/\r?\n/).map((line) => line.trim().replace(/^\|/, "").replace(/\|$/, "").split("|").map((cell) => cell.trim()));
  if (rows.length < 2 || !rows[1].every((cell) => /^:?-{3,}:?$/.test(cell))) return tableBlock;
  const header = rows[0].map((cell) => `<th>${cell}</th>`).join("");
  const body = rows.slice(2).map((row) => `<tr>${row.map((cell) => `<td>${cell}</td>`).join("")}</tr>`).join("");
  return `<table><thead><tr>${header}</tr></thead><tbody>${body}</tbody></table>`;
}
