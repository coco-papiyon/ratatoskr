import type { ExplorerNode } from "./types";

export const demoNodes: ExplorerNode[] = [
  { id: "readme", name: "README.md", kind: "file", path: "/README.md", sourceId: "demo", size: 1840, modifiedAt: "2026-07-14 10:22" },
  { id: "architecture", name: "architecture.md", kind: "file", path: "/docs/architecture.md", sourceId: "demo", size: 4200, modifiedAt: "2026-07-13 18:14" },
  { id: "sky", name: "northern-lights.jpg", kind: "file", path: "/assets/northern-lights.jpg", sourceId: "demo", size: 2800000, modifiedAt: "2026-07-12 09:40", mimeType: "image/jpeg" },
  { id: "config", name: "viewer.config.json", kind: "file", path: "/viewer.config.json", sourceId: "demo", size: 356, modifiedAt: "2026-07-11 14:02" },
];

export const demoMarkdown = `# Ratatoskr workspace

複数のストレージを、ひとつの落ち着いた作業面で扱うためのビューアです。

## 今日の確認

- S3 プロファイルを選択
- フォルダをたどる
- ファイルをその場で確認

> 左のタブで接続先を切り替えられます。

\`\`\`ts
const source = await explorer.open("local");
\`\`\`
`;
