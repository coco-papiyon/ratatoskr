export interface PreviewItem {
  name: string;
  kind: "file" | "folder";
}

export function isPreviewable(item: PreviewItem): boolean {
  return item.kind === "file";
}
