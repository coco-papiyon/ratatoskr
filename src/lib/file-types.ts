import type { ExplorerNode, PreviewData, StructuredFormat, ViewerConfig } from "../types";

export const archiveExtensions = [".zip", ".tar", ".tar.gz", ".tgz"] as const;

export function isArchiveName(name: string) {
  const lower = name.toLowerCase();
  return archiveExtensions.some((extension) => lower.endsWith(extension));
}

export function matchesConfiguredExtension(name: string, category: keyof ViewerConfig["extensions"], config: ViewerConfig) {
  return config.extensions[category]?.some((pattern) => name === pattern.toLowerCase() || (pattern.startsWith(".") && name.endsWith(pattern.toLowerCase()))) ?? false;
}

export function previewKind(node: ExplorerNode, config: ViewerConfig): PreviewData["kind"] {
  const name = node.name.toLowerCase();
  if (matchesConfiguredExtension(name, "markdown", config)) return "markdown";
  if (matchesConfiguredExtension(name, "image", config)) return "image";
  if (matchesConfiguredExtension(name, "structured", config)) return "structured";
  if (matchesConfiguredExtension(name, "text", config)) return "text";
  return "unsupported";
}

export function structuredFormat(node: Pick<ExplorerNode, "name">): StructuredFormat | undefined {
  const name = node.name.toLowerCase();
  if (name.endsWith(".json")) return "json";
  if (name.endsWith(".xml")) return "xml";
  if (name.endsWith(".csv")) return "csv";
  if (name.endsWith(".yaml") || name.endsWith(".yml")) return "yaml";
  return undefined;
}

export function parseS3Path(path: string) {
  if (!path.startsWith("s3://")) return undefined;
  const [bucket, ...keyParts] = path.slice(5).split("/");
  return bucket ? { bucket, key: keyParts.join("/") } : undefined;
}

export function formatSize(size?: number) {
  if (!size) return "--";
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${Math.round(size / 1024)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
}
