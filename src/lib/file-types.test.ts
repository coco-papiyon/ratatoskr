import { describe, expect, it } from "vitest";
import { formatSize, isArchiveName, parseS3Path, previewKind, structuredFormat } from "./file-types";
import type { ExplorerNode, ViewerConfig } from "../types";

const config: ViewerConfig = {
  extensions: {
    markdown: [".md"],
    image: [".png", ".svg"],
    structured: [".json", ".yaml", ".yml", ".csv"],
    text: [".txt", ".log", ".gitignore"],
  },
};

function node(name: string): ExplorerNode {
  return { id: name, name, kind: "file", path: `/data/${name}`, sourceId: "local" };
}

describe("file type detection", () => {
  it.each([
    ["README.md", "markdown"],
    ["image.svg", "image"],
    ["records.json", "structured"],
    ["application.log", "text"],
    ["binary.exe", "unsupported"],
  ])("classifies %s as %s", (name, expected) => {
    expect(previewKind(node(name), config)).toBe(expected);
  });

  it.each(["sample.zip", "sample.tar", "sample.tar.gz", "sample.tgz", "SAMPLE.ZIP"])("recognizes archive %s", (name) => {
    expect(isArchiveName(name)).toBe(true);
  });

  it("detects YAML structured formats", () => {
    expect(structuredFormat(node("config.yaml"))).toBe("yaml");
    expect(structuredFormat(node("config.yml"))).toBe("yaml");
  });
});

describe("path and size formatting", () => {
  it("splits an S3 URI into bucket and key", () => {
    expect(parseS3Path("s3://archive-bucket/folder/sample.zip")).toEqual({ bucket: "archive-bucket", key: "folder/sample.zip" });
    expect(parseS3Path("C:/data/file.txt")).toBeUndefined();
  });

  it("formats common file sizes", () => {
    expect(formatSize()).toBe("--");
    expect(formatSize(512)).toBe("512 B");
    expect(formatSize(2048)).toBe("2 KB");
    expect(formatSize(1572864)).toBe("1.5 MB");
  });
});
