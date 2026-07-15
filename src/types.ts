export type SourceType = "local" | "s3";
export type NodeKind = "folder" | "file";
export type StructuredFormat = "json" | "xml" | "csv" | "yaml";
export type PreviewKind = "markdown" | "image" | "text" | "structured" | "unsupported";
export type PreviewCategory = "markdown" | "image" | "text" | "structured";

export interface ExplorerNode {
  id: string;
  name: string;
  kind: NodeKind;
  path: string;
  sourceId: string;
  modifiedAt?: string;
  size?: number;
  mimeType?: string;
  archivePath?: string;
  archiveEntry?: string;
  handle?: FileSystemFileHandle | FileSystemDirectoryHandle;
}

export interface PreviewData {
  kind: PreviewKind;
  content: string;
  url?: string;
  format?: StructuredFormat;
}

export interface ViewerConfig {
  extensions: Record<PreviewCategory, string[]>;
  proxy: string;
  certificate: string;
}

export interface StructuredTableRule {
  name: string;
  filePattern: string;
  jq: string;
}

declare global {
  interface FileSystemDirectoryHandle {
    entries(): AsyncIterableIterator<[string, FileSystemHandle]>;
  }

  interface Window {
    showDirectoryPicker(): Promise<FileSystemDirectoryHandle>;
  }
}
