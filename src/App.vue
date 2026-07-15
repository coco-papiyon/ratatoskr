<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import SettingsDialog from "./components/SettingsDialog.vue";
import { archiveExtensions, formatSize, isArchiveName, parseS3Path, previewKind as resolvePreviewKind, structuredFormat } from "./lib/file-types";
import { ansiToHtml, highlightCode, markdownToHtml, parseCsv } from "./lib/viewer-format";
import { demoMarkdown } from "./mock-data";
import type { ExplorerNode, PreviewData, SourceType, StructuredFormat, StructuredTableRule, ViewerConfig } from "./types";

const activeSource = ref<SourceType>("local");
const headerVisible = ref(true);
const initialLocalNode: ExplorerNode = { id: "local-root", name: "Current directory", kind: "folder", path: "", sourceId: "local" };
const selectedFolder = ref("");
const nodes = ref<ExplorerNode[]>([]);
const localNodes = ref<ExplorerNode[]>([]);
const localFolder = ref("");
const selectedNode = ref<ExplorerNode>(initialLocalNode);
const localSelectedNode = ref<ExplorerNode>(initialLocalNode);
const query = ref("");
const loading = ref(false);
const error = ref("");
const directoryHandle = ref<FileSystemDirectoryHandle>();
const currentPath = ref("");
const preview = ref<PreviewData>({ kind: "unsupported", content: "現在のディレクトリを読み込んでいます。" });
const structuredTable = ref<StructuredTable | null>(null);
const structuredViewMode = ref<"table" | "source">("table");
const listWidth = ref(30);
const resizing = ref(false);
const resizeContainer = ref<HTMLElement>();
const awsProfiles = ref(["default"]);
const awsProfile = ref("default");
const awsRegion = ref("ap-northeast-1");
const selectedEncoding = ref("auto");
const settingsVisible = ref(false);
const settingsError = ref("");
const settingsDraft = ref<ViewerConfig>({ extensions: {
  markdown: [], text: [], image: [], structured: [],
}});
const structuredRulesDraft = ref<StructuredTableRule[]>([]);
const archiveContext = ref<ArchiveContext | null>(null);
const localArchiveContext = ref<ArchiveContext | null>(null);
const favorites = ref<ExplorerNode[]>(loadStoredNodes("ratatoskr.favorites"));
const recent = ref<ExplorerNode[]>(loadStoredNodes("ratatoskr.recent"));
const viewerConfig = ref<ViewerConfig>({ extensions: {
  markdown: [".md", ".markdown", ".mdx"],
  text: [".txt", ".log", ".out", ".err", ".yaml", ".yml", ".toml", ".ini", ".conf", ".properties", ".lock", ".mod", ".sum", ".md5", ".patch", ".diff", ".map", ".ts", ".tsx", ".vue", ".js", ".jsx", ".css", ".scss", ".html", ".go", ".rs", ".py", ".java", ".c", ".cpp", ".h", ".sql", ".sh", ".ps1", ".bat", ".gitignore", ".gitattributes", ".gitmodules", ".dockerignore", ".editorconfig", ".env", "dockerfile", "makefile", "procfile", "license"],
  image: [".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".avif"],
  structured: [".json", ".xml", ".csv", ".yaml", ".yml"],
}});
let objectUrl: string | undefined;

const filteredNodes = computed(() => {
  const needle = query.value.trim().toLowerCase();
  return needle ? nodes.value.filter((node) => node.name.toLowerCase().includes(needle)) : nodes.value;
});

const breadcrumbs = computed(() => activeSource.value === "local" ? currentPath.value || "Current directory" : selectedNode.value.path);
const canNavigateUp = computed(() => {
  if (!desktopBridge()) return false;
  return activeSource.value === "local" ? currentPath.value !== "" : selectedNode.value.path !== "s3://";
});
const currentDirectoryName = computed(() => {
  const path = currentPath.value.replace(/[\\/]$/, "");
  const parts = path.split(/[\\/]/).filter(Boolean);
  return parts[parts.length - 1] ?? "Local";
});
const highlightedPreview = computed(() => {
  if (preview.value.kind !== "text") return "";
  return /\x1b\[[0-9;]*m/.test(preview.value.content) ? ansiToHtml(preview.value.content) : highlightCode(preview.value.content);
});
const structuredText = computed(() => {
  if (preview.value.kind !== "structured" || preview.value.format === "csv") return "";
  if (preview.value.format === "json") {
    try {
      return JSON.stringify(JSON.parse(preview.value.content), null, 2);
    } catch {
      return preview.value.content;
    }
  }
  return preview.value.content.replace(/></g, ">\n<");
});
const csvHeaderEnabled = ref(true);
const csvRows = computed(() => preview.value.kind === "structured" && preview.value.format === "csv" ? parseCsv(preview.value.content) : []);
const csvColumns = computed(() => {
  const rows = csvRows.value;
  const width = Math.max(...rows.map((row) => row.length), 0);
  return Array.from({ length: width }, (_, index) => `Column ${index + 1}`);
});
const csvHeader = computed(() => csvHeaderEnabled.value && csvRows.value.length ? csvRows.value[0] : csvColumns.value);
const csvBody = computed(() => csvHeaderEnabled.value ? csvRows.value.slice(1) : csvRows.value);

type WailsBridge = {
  CurrentWorkingDirectory(): Promise<string>;
  ParentLocalDirectory(path: string): Promise<string>;
  SelectLocalDirectory(): Promise<string>;
  ListLocalDirectory(path: string): Promise<ExplorerNode[]>;
  ReadLocalTextFile(path: string): Promise<string>;
  ReadLocalPreview(path: string, charset: string): Promise<S3Preview>;
  ListLocalArchive(archivePath: string, prefix: string): Promise<ExplorerNode[]>;
  ReadLocalArchivePreview(archivePath: string, entryPath: string, charset: string): Promise<S3Preview>;
  ListAWSProfiles(): Promise<string[]>;
  ListS3Buckets(profile: string, region: string): Promise<ExplorerNode[]>;
  ListS3Directory(profile: string, region: string, bucket: string, prefix: string): Promise<ExplorerNode[]>;
  ReadS3Preview(profile: string, region: string, bucket: string, key: string, charset: string): Promise<S3Preview>;
  ListS3Archive(profile: string, region: string, bucket: string, key: string, prefix: string): Promise<ExplorerNode[]>;
  ReadS3ArchivePreview(profile: string, region: string, bucket: string, key: string, entryPath: string, charset: string): Promise<S3Preview>;
  GetViewerConfig(): Promise<ViewerConfig>;
  UpdateViewerConfig(config: ViewerConfig): Promise<void>;
  GetStructuredTableRules(): Promise<StructuredTableRule[]>;
  UpdateStructuredTableRules(rules: StructuredTableRule[]): Promise<void>;
  ConvertStructuredToTable(filePath: string, content: string): Promise<StructuredTable | null>;
};

type S3Preview = { content: string; dataUrl?: string };
type StructuredTable = { ruleName: string; columns: string[]; rows: string[][] };
type ArchiveContext = { source: SourceType; archivePath: string; prefix: string };

function desktopBridge(): WailsBridge | undefined {
  const go = (window as unknown as { go?: { app?: { App?: WailsBridge }; main?: { App?: WailsBridge } } }).go;
  return go?.app?.App ?? go?.main?.App;
}

function loadStoredNodes(key: string): ExplorerNode[] {
  if (typeof window === "undefined") return [];
  try {
    const value = JSON.parse(window.localStorage.getItem(key) ?? "[]") as ExplorerNode[];
    return Array.isArray(value) ? value : [];
  } catch {
    return [];
  }
}

function persistStoredNodes(key: string, value: ExplorerNode[]) {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // Storage may be disabled by the embedded browser policy.
  }
}

function storedNode(node: ExplorerNode): ExplorerNode {
  const { handle: _handle, ...withoutHandle } = node;
  return { ...withoutHandle, sourceId: activeSource.value };
}

function nodeKey(node: ExplorerNode) {
  return `${node.sourceId}:${node.archivePath ?? node.path}:${node.archiveEntry ?? ""}`;
}

function isFavorite(node: ExplorerNode) {
  return favorites.value.some((favorite) => nodeKey(favorite) === nodeKey(storedNode(node)));
}

function toggleFavorite(node: ExplorerNode) {
  const saved = storedNode(node);
  const key = nodeKey(saved);
  const index = favorites.value.findIndex((favorite) => nodeKey(favorite) === key);
  if (index >= 0) favorites.value.splice(index, 1);
  else favorites.value.unshift(saved);
  persistStoredNodes("ratatoskr.favorites", favorites.value);
}

function addRecent(node: ExplorerNode) {
  if (node.kind !== "file") return;
  const saved = storedNode(node);
  recent.value = [saved, ...recent.value.filter((item) => nodeKey(item) !== nodeKey(saved))].slice(0, 8);
  persistStoredNodes("ratatoskr.recent", recent.value);
}

async function openStoredNode(node: ExplorerNode) {
  activeSource.value = node.sourceId === "s3" ? "s3" : "local";
  await selectNode({ ...node, handle: undefined });
}

function previewKind(node: ExplorerNode): PreviewData["kind"] {
  return resolvePreviewKind(node, viewerConfig.value);
}

async function selectNode(node: ExplorerNode) {
  addRecent(node);
  selectedNode.value = node;
  if (activeSource.value === "local") localSelectedNode.value = node;
  error.value = "";
  structuredTable.value = null;
  if (objectUrl) URL.revokeObjectURL(objectUrl);
  objectUrl = undefined;

  if (node.archivePath) {
    if (node.kind === "folder") {
      await openArchiveDirectory(node.archivePath, node.archiveEntry ?? "");
    } else if (node.archiveEntry) {
      await readArchiveNode(node);
    }
    return;
  }

  if (node.kind === "file" && isArchiveName(node.name) && desktopBridge() && !node.handle) {
    await openArchiveDirectory(node.path, "");
    return;
  }

  if (activeSource.value === "s3") {
    if (node.kind === "folder") {
      const location = parseS3Path(node.path);
      if (location) await openS3Directory(location.bucket, location.key);
      return;
    }
    const location = parseS3Path(node.path);
    const bridge = desktopBridge();
    if (!location || !bridge) return;
    try {
      const kind = previewKind(node);
      const data = await bridge.ReadS3Preview(awsProfile.value, awsRegion.value, location.bucket, location.key, selectedEncoding.value);
      preview.value = kind === "image"
        ? { kind, content: "", url: data.dataUrl }
        : kind === "markdown" || kind === "text"
          ? { kind, content: data.content }
          : kind === "structured"
            ? { kind, format: structuredFormat(node), content: data.content }
          : { kind, content: "この形式はプレビューに対応していません。" };
      if (kind === "structured") await prepareStructuredTable(node, data.content);
    } catch {
      error.value = "S3 オブジェクトを読み込めませんでした。";
    }
    return;
  }

  if (node.kind === "folder") {
    if (activeSource.value === "local" && desktopBridge()) {
      await openLocalDirectory(node.path);
      return;
    }
    if (node.handle) {
      await openBrowserDirectory(node.handle as FileSystemDirectoryHandle, node.path);
      return;
    }
  }

  const bridge = desktopBridge();
  if (activeSource.value === "local" && bridge && !node.handle) {
    try {
      const kind = previewKind(node);
      const data = await bridge.ReadLocalPreview(node.path, selectedEncoding.value);
      preview.value = kind === "image"
        ? { kind, content: "", url: data.dataUrl }
        : kind === "markdown" || kind === "text" || kind === "structured"
          ? { kind, format: kind === "structured" ? structuredFormat(node) : undefined, content: data.content }
          : { kind, content: "この形式はプレビューに対応していません。" };
      if (kind === "structured") await prepareStructuredTable(node, data.content);
    } catch {
      error.value = "ファイルの読み込みに失敗しました。";
    }
    return;
  }

  if (!node.handle) {
    preview.value = node.name === "README.md"
      ? { kind: "markdown", content: demoMarkdown }
      : { kind: previewKind(node), format: previewKind(node) === "structured" ? structuredFormat(node) : undefined, content: node.name === "architecture.md" ? demoMarkdown : '{\n  "headerVisible": true,\n  "defaultSource": "local"\n}' };
    return;
  }

  try {
    const file = await (node.handle as FileSystemFileHandle).getFile();
    const kind = previewKind(node);
    if (kind === "image") {
      objectUrl = URL.createObjectURL(file);
      preview.value = { kind, content: "", url: objectUrl };
    } else if (kind === "markdown" || kind === "text" || kind === "structured") {
      const content = await file.text();
      preview.value = { kind, format: kind === "structured" ? structuredFormat(node) : undefined, content };
      if (kind === "structured") await prepareStructuredTable(node, content);
    } else {
      preview.value = { kind, content: "この形式はプレビューに対応していません。" };
    }
  } catch {
    error.value = "ファイルの読み込みに失敗しました。もう一度選択してください。";
  }
}

async function prepareStructuredTable(node: ExplorerNode, content: string) {
  structuredTable.value = null;
  if (!(["json", "yaml"] as Array<StructuredFormat | undefined>).includes(structuredFormat(node))) return;
  const bridge = desktopBridge();
  if (!bridge) return;
  try {
    structuredTable.value = await bridge.ConvertStructuredToTable(node.path, content);
    if (structuredTable.value) structuredViewMode.value = "table";
  } catch {
    error.value = "構造化データの表変換に失敗しました。config/json-table.yaml のルールを確認してください。";
  }
}

async function openArchiveDirectory(archivePath: string, prefix: string) {
  const bridge = desktopBridge();
  if (!bridge) return;
  try {
    loading.value = true;
    error.value = "";
    const context: ArchiveContext = { source: activeSource.value, archivePath, prefix };
    const entries = activeSource.value === "s3"
      ? await listS3ArchiveDirectory(archivePath, prefix)
      : await bridge.ListLocalArchive(archivePath, prefix);
    const virtualPath = `${archivePath}!/${prefix}`;
    archiveContext.value = context;
    nodes.value = entries;
    selectedFolder.value = archivePath;
    selectedNode.value = {
      id: virtualPath,
      name: prefix.replace(/\/$/, "").split("/").pop() || archivePath.split(/[\\/]/).pop() || archivePath,
      kind: "folder",
      path: virtualPath,
      sourceId: activeSource.value,
      archivePath,
      archiveEntry: prefix || undefined,
    };
    preview.value = { kind: "unsupported", content: "圧縮ファイル内のフォルダを選択しています。左の一覧からファイルを選んでください。" };
    if (activeSource.value === "local") {
      currentPath.value = virtualPath;
      localFolder.value = archivePath;
      localNodes.value = entries;
      localSelectedNode.value = selectedNode.value;
      localArchiveContext.value = context;
    }
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : `圧縮ファイルを開けませんでした: ${String(caught)}`;
  } finally {
    loading.value = false;
  }
}

async function listS3ArchiveDirectory(archivePath: string, prefix: string) {
  const bridge = desktopBridge();
  const location = parseS3Path(archivePath);
  if (!bridge || !location) throw new Error("S3の圧縮ファイルパスが不正です。");
  return bridge.ListS3Archive(awsProfile.value, awsRegion.value, location.bucket, location.key, prefix);
}

async function readArchiveNode(node: ExplorerNode) {
  const bridge = desktopBridge();
  if (!bridge || !node.archivePath || !node.archiveEntry) return;
  try {
    const kind = previewKind(node);
    const data = activeSource.value === "s3"
      ? await readS3ArchiveNode(node.archivePath, node.archiveEntry)
      : await bridge.ReadLocalArchivePreview(node.archivePath, node.archiveEntry, selectedEncoding.value);
    preview.value = kind === "image"
      ? { kind, content: "", url: data.dataUrl }
      : kind === "markdown" || kind === "text"
        ? { kind, content: data.content }
        : kind === "structured"
          ? { kind, format: structuredFormat(node), content: data.content }
          : { kind, content: "この形式はプレビューに対応していません。" };
    if (kind === "structured") await prepareStructuredTable(node, data.content);
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : `圧縮ファイル内の項目を読み込めませんでした: ${String(caught)}`;
  }
}

async function readS3ArchiveNode(archivePath: string, entryPath: string) {
  const bridge = desktopBridge();
  const location = parseS3Path(archivePath);
  if (!bridge || !location) throw new Error("S3の圧縮ファイルパスが不正です。");
  return bridge.ReadS3ArchivePreview(awsProfile.value, awsRegion.value, location.bucket, location.key, entryPath, selectedEncoding.value);
}

async function chooseFolder() {
  const bridge = desktopBridge();
  if (bridge) {
    try {
      const path = await bridge.SelectLocalDirectory();
      if (path) await openLocalDirectory(path);
    } catch {
      error.value = "フォルダを開けませんでした。権限を確認してください。";
    }
    return;
  }
  if (!("showDirectoryPicker" in window)) {
    error.value = "このブラウザはローカルフォルダの選択に対応していません。Chromium 系ブラウザで開いてください。";
    return;
  }
  try {
    loading.value = true;
    error.value = "";
    const handle = await window.showDirectoryPicker();
    await openBrowserDirectory(handle, `/${handle.name}`);
  } catch (caught) {
    if ((caught as DOMException).name !== "AbortError") error.value = "フォルダを開けませんでした。権限を確認してください。";
  }
}

async function openLocalDirectory(path: string) {
  const bridge = desktopBridge();
  if (!bridge) return;
  try {
    loading.value = true;
    error.value = "";
    const entries = await bridge.ListLocalDirectory(path);
    archiveContext.value = null;
    localArchiveContext.value = null;
    currentPath.value = path;
    selectedFolder.value = path;
    nodes.value = entries;
    localFolder.value = path;
    localNodes.value = entries;
    selectedNode.value = { id: path, name: path, kind: "folder", path, sourceId: "local" };
    localSelectedNode.value = selectedNode.value;
    preview.value = { kind: "unsupported", content: "フォルダを選択しています。左の一覧からファイルを選んでください。" };
  } catch {
    error.value = "フォルダを読み込めませんでした。権限を確認してください。";
  } finally {
    loading.value = false;
  }
}

async function goToParentDirectory() {
  const bridge = desktopBridge();
  if (!bridge) return;
  const activeArchive = archiveContext.value;
  if (activeArchive && activeArchive.source === activeSource.value) {
    const cleanPrefix = activeArchive.prefix.replace(/\/$/, "");
    if (cleanPrefix) {
      const separator = cleanPrefix.lastIndexOf("/");
      await openArchiveDirectory(activeArchive.archivePath, separator < 0 ? "" : cleanPrefix.slice(0, separator + 1));
      return;
    }
    if (activeSource.value === "local") {
      const parentPath = await bridge.ParentLocalDirectory(activeArchive.archivePath);
      await openLocalDirectory(parentPath);
      return;
    }
    const location = parseS3Path(activeArchive.archivePath);
    if (location) {
      const separator = location.key.lastIndexOf("/");
      await openS3Directory(location.bucket, separator < 0 ? "" : location.key.slice(0, separator + 1));
    }
    return;
  }
  if (activeSource.value === "s3") {
    const location = parseS3Path(selectedNode.value.path);
    if (!location) return;
    if (!location.key) {
      await openS3Buckets();
      return;
    }
    const key = location.key.replace(/\/$/, "");
    const separator = key.lastIndexOf("/");
    await openS3Directory(location.bucket, separator < 0 ? "" : key.slice(0, separator + 1));
    return;
  }
  try {
    const parentPath = await bridge.ParentLocalDirectory(currentPath.value);
    await openLocalDirectory(parentPath);
  } catch {
    error.value = "これ以上上のディレクトリへは移動できません。";
  }
}

async function openBrowserDirectory(handle: FileSystemDirectoryHandle, path: string) {
  try {
    loading.value = true;
    error.value = "";
    directoryHandle.value = handle;
    selectedFolder.value = handle.name;
    currentPath.value = path;
    const entries: ExplorerNode[] = [];
    for await (const [name, entryHandle] of handle.entries()) {
      const isFile = entryHandle.kind === "file";
      let size: number | undefined;
      let modifiedAt: string | undefined;
      if (isFile) {
        const file = await (entryHandle as FileSystemFileHandle).getFile();
        size = file.size;
        modifiedAt = new Intl.DateTimeFormat("ja-JP", { dateStyle: "medium", timeStyle: "short" }).format(file.lastModified);
      }
      entries.push({ id: `${path}/${name}`, name, kind: isFile ? "file" : "folder", path: `${path}/${name}`, sourceId: "local", handle: entryHandle as FileSystemFileHandle | FileSystemDirectoryHandle, size, modifiedAt, mimeType: isFile ? (entryHandle as FileSystemFileHandle).name : undefined });
    }
    nodes.value = entries.sort((a, b) => Number(b.kind === "folder") - Number(a.kind === "folder") || a.name.localeCompare(b.name));
    localFolder.value = handle.name;
    localNodes.value = nodes.value;
    selectedNode.value = { id: path, name: handle.name, kind: "folder", path, sourceId: "local", handle };
    localSelectedNode.value = selectedNode.value;
    preview.value = { kind: "unsupported", content: "フォルダを選択しています。左の一覧からファイルを選んでください。" };
  } catch (caught) {
    if ((caught as DOMException).name !== "AbortError") error.value = "フォルダを開けませんでした。権限を確認してください。";
  } finally {
    loading.value = false;
  }
}

function switchSource(source: SourceType) {
  activeSource.value = source;
  error.value = "";
  if (source === "local") {
    archiveContext.value = localArchiveContext.value;
    selectedFolder.value = localFolder.value;
    nodes.value = localNodes.value;
    selectedNode.value = localSelectedNode.value;
    void selectNode(localSelectedNode.value);
  } else {
    void openS3Buckets();
  }
}

async function openS3Buckets() {
  const bridge = desktopBridge();
  if (!bridge) return;
  try {
    loading.value = true;
    error.value = "";
    const entries = await bridge.ListS3Buckets(awsProfile.value, awsRegion.value);
    archiveContext.value = null;
    nodes.value = entries;
    selectedFolder.value = "S3 buckets";
    selectedNode.value = { id: "s3://", name: "S3 buckets", kind: "folder", path: "s3://", sourceId: "s3" };
    preview.value = { kind: "markdown", content: "# S3 Explorer\n\nバケットを選択して内容を表示します。" };
  } catch {
    error.value = "S3 バケット一覧を取得できませんでした。AWS Profile、リージョン、認証情報を確認してください。";
    nodes.value = [];
  } finally {
    loading.value = false;
  }
}

async function openS3Directory(bucket: string, prefix: string) {
  const bridge = desktopBridge();
  if (!bridge) return;
  try {
    loading.value = true;
    error.value = "";
    const entries = await bridge.ListS3Directory(awsProfile.value, awsRegion.value, bucket, prefix);
    archiveContext.value = null;
    nodes.value = entries;
    selectedFolder.value = bucket;
    const path = `s3://${bucket}${prefix ? `/${prefix}` : ""}`;
    selectedNode.value = { id: path, name: prefix || bucket, kind: "folder", path, sourceId: "s3" };
    preview.value = { kind: "unsupported", content: "フォルダを選択しています。左の一覧からファイルを選んでください。" };
  } catch {
    error.value = "S3 の一覧を取得できませんでした。アクセス権限を確認してください。";
  } finally {
    loading.value = false;
  }
}

function refreshActiveSource() {
  if (activeSource.value === "s3") {
    void openS3Buckets();
    return;
  }
  if (desktopBridge()) void openLocalDirectory(currentPath.value);
}

function reloadSelectedFile() {
  if (selectedNode.value.kind === "file") void selectNode(selectedNode.value);
}

async function openSettings() {
  settingsDraft.value = { extensions: Object.fromEntries(Object.entries(viewerConfig.value.extensions).map(([category, extensions]) => [category, [...extensions]])) as ViewerConfig["extensions"] };
  structuredRulesDraft.value = [];
  settingsError.value = "";
  settingsVisible.value = true;
  const bridge = desktopBridge();
  if (!bridge) return;
  try {
    structuredRulesDraft.value = (await bridge.GetStructuredTableRules()).map((rule) => ({ ...rule }));
  } catch {
    settingsError.value = "表変換設定を読み込めませんでした。";
  }
}

async function saveSettings() {
  const bridge = desktopBridge();
  if (!bridge) {
    viewerConfig.value = settingsDraft.value;
    settingsVisible.value = false;
    return;
  }
  try {
    settingsError.value = "";
    await bridge.UpdateStructuredTableRules(structuredRulesDraft.value);
    await bridge.UpdateViewerConfig(settingsDraft.value);
    viewerConfig.value = settingsDraft.value;
    settingsVisible.value = false;
    error.value = "";
  } catch (caught) {
    settingsError.value = caught instanceof Error ? caught.message : `設定を保存できませんでした: ${String(caught)}`;
  }
}

function resizeList(event: PointerEvent) {
  const bodyGrid = resizeContainer.value;
  if (!bodyGrid) return;
  const rect = bodyGrid.getBoundingClientRect();
  listWidth.value = Math.min(42, Math.max(22, ((event.clientX - rect.left) / rect.width) * 100));
}

function stopResize() {
  if (!resizing.value) return;
  resizing.value = false;
  window.removeEventListener("pointermove", resizeList);
  window.removeEventListener("pointerup", stopResize);
}

function startResize(event: PointerEvent) {
  resizing.value = true;
  resizeContainer.value = (event.currentTarget as HTMLElement).parentElement ?? undefined;
  (event.currentTarget as HTMLElement).setPointerCapture?.(event.pointerId);
  window.addEventListener("pointermove", resizeList);
  window.addEventListener("pointerup", stopResize);
}

onBeforeUnmount(() => {
  if (objectUrl) URL.revokeObjectURL(objectUrl);
  stopResize();
});

onMounted(async () => {
  const bridge = desktopBridge();
  if (!bridge) return;
  try {
    viewerConfig.value = await bridge.GetViewerConfig();
    awsProfiles.value = await bridge.ListAWSProfiles();
    if (!awsProfiles.value.includes(awsProfile.value)) awsProfile.value = awsProfiles.value[0] ?? "default";
    await openLocalDirectory(await bridge.CurrentWorkingDirectory());
  } catch {
    error.value = "カレントディレクトリを開けませんでした。";
  }
});
</script>

<template>
  <main class="app-shell" :class="{ 'header-hidden': !headerVisible }">
    <header v-if="headerVisible" class="app-header">
      <div class="brand"><span class="brand-mark">R</span><span>ratatoskr</span></div>
      <nav class="header-source-tabs" aria-label="ソース"><button :class="{ active: activeSource === 'local' }" @click="switchSource('local')">⌂ Local</button><button :class="{ active: activeSource === 's3' }" @click="switchSource('s3')">☁ S3</button></nav>
      <div class="header-actions"><button class="icon-button" title="設定" @click="openSettings">⚙</button><button class="icon-button" title="ヘッダーを隠す" @click="headerVisible = false">⌃</button><button class="primary-button" @click="refreshActiveSource">↻ 更新</button></div>
    </header>

    <section class="workspace">
      <aside class="source-pane">
        <div v-if="!headerVisible" class="compact-top"><div class="brand"><span class="brand-mark">R</span><span>ratatoskr</span></div><button class="text-button" @click="headerVisible = true">ヘッダーを表示</button></div>
        <div class="source-content">
          <div v-if="activeSource === 's3'" class="s3-settings">
            <p class="eyebrow">S3 CONNECTION</p>
            <label class="settings-field">AWS Profile<select v-model="awsProfile" @change="refreshActiveSource"><option v-for="profile in awsProfiles" :key="profile" :value="profile">{{ profile }}</option></select></label>
            <label class="settings-field">Region<select v-model="awsRegion" @change="refreshActiveSource"><option>ap-northeast-1</option><option>us-east-1</option><option>us-west-2</option><option>eu-west-1</option></select></label>
            <div class="connection"><i></i> Ready</div>
          </div>
          <p class="eyebrow">{{ activeSource === 'local' ? 'LOCATION' : 'BUCKET' }}</p>
          <button class="location-card" :class="{ selected: activeSource === 'local' }" @click="activeSource === 'local' ? chooseFolder() : openS3Buckets()"><span class="location-icon">{{ activeSource === 'local' ? '⌘' : '◒' }}</span><span><strong>{{ activeSource === 'local' ? currentDirectoryName : selectedFolder }}</strong><small>{{ activeSource === 'local' ? 'ローカルフォルダを開く' : `${awsProfile} / ${awsRegion}` }}</small></span></button>
          <p class="eyebrow">FAVORITES</p>
          <button v-for="node in favorites" :key="`favorite-${nodeKey(node)}`" class="nav-item stored-nav-item" @click="openStoredNode(node)"><span>★</span><span>{{ node.name }}</span></button>
          <p v-if="!favorites.length" class="stored-empty">一覧の星から追加できます</p>
          <p class="eyebrow">RECENT</p>
          <button v-for="node in recent" :key="`recent-${nodeKey(node)}`" class="nav-item stored-nav-item" @click="openStoredNode(node)"><span>◷</span><span>{{ node.name }}</span></button>
          <p v-if="!recent.length" class="stored-empty">選択したファイルが表示されます</p>
        </div>
        <div class="pane-footer"><span>● {{ activeSource === 'local' ? 'Local access' : 'S3 setup required' }}</span></div>
      </aside>

      <section class="content-pane">
        <div class="content-toolbar"><div class="path-controls"><button v-if="canNavigateUp" class="up-button" title="上のディレクトリへ移動" aria-label="上のディレクトリへ移動" @click="goToParentDirectory">↑</button><div class="breadcrumbs" :title="breadcrumbs"><span>{{ breadcrumbs }}</span></div></div><label class="search"><span>⌕</span><input v-model="query" placeholder="ファイルを検索" /></label></div>
        <p v-if="error" class="error-message">{{ error }}</p>
        <div class="body-grid" :style="{ '--list-width': `${listWidth}%` }" :class="{ resizing }">
          <section class="file-list" aria-label="ファイル一覧">
            <div class="list-heading"><span>Files</span><span>{{ filteredNodes.length }} items</span></div>
            <button v-for="node in filteredNodes" :key="node.id" class="file-row" :class="{ selected: selectedNode.id === node.id }" @click="selectNode(node)"><span class="file-icon">{{ node.kind === 'folder' ? '□' : isArchiveName(node.name) ? '▣' : previewKind(node) === 'image' ? '▧' : previewKind(node) === 'markdown' ? '◆' : '▤' }}</span><span class="file-name">{{ node.name }}<small>{{ node.modifiedAt ?? node.path }}</small></span><span class="file-size">{{ formatSize(node.size) }}</span><span class="favorite-toggle" :class="{ active: isFavorite(node) }" role="button" tabindex="0" :aria-label="isFavorite(node) ? 'お気に入りから削除' : 'お気に入りに追加'" @click.stop="toggleFavorite(node)" @keydown.enter.stop="toggleFavorite(node)">{{ isFavorite(node) ? '★' : '☆' }}</span></button>
            <div v-if="loading" class="empty-state">フォルダを読み込んでいます…</div>
            <div v-else-if="!filteredNodes.length && activeSource === 's3'" class="empty-state">AWS プロファイルを選択して S3 接続を開始します。</div>
            <div v-else-if="!filteredNodes.length" class="empty-state">一致するファイルがありません。</div>
          </section>
          <div class="splitter" role="separator" aria-label="一覧と本文の幅を変更" title="ドラッグして幅を変更" @pointerdown="startResize"><span></span></div>
          <section class="viewer-pane">
            <div class="viewer-toolbar"><span class="viewer-title"><i :class="`type-${preview.kind}`"></i>{{ selectedNode.name }}</span><span class="viewer-controls"><button v-if="structuredTable" class="structured-toggle" @click="structuredViewMode = structuredViewMode === 'table' ? 'source' : 'table'">{{ structuredViewMode === 'table' ? 'Source' : 'Table' }}</button><label>Encoding <select v-model="selectedEncoding" @change="reloadSelectedFile"><option value="auto">Auto</option><option value="utf-8">UTF-8</option><option value="shift-jis">Shift_JIS</option><option value="euc-jp">EUC-JP</option><option value="iso-2022-jp">ISO-2022-JP</option></select></label><span>{{ preview.kind }}</span></span></div>
            <article v-if="preview.kind === 'markdown'" class="markdown-view" v-html="markdownToHtml(preview.content)" />
            <div v-else-if="preview.kind === 'image'" class="image-view"><img :src="preview.url" :alt="selectedNode.name" /><span>Fit to view</span></div>
            <pre v-else-if="preview.kind === 'text'" class="text-view"><code v-html="highlightedPreview" /></pre>
            <section v-else-if="preview.kind === 'structured' && preview.format === 'csv'" class="csv-view">
              <div class="structured-toolbar"><span>CSV table</span><label><input v-model="csvHeaderEnabled" type="checkbox" /> 先頭行をヘッダーとして扱う</label></div>
              <div class="csv-scroll"><table><thead><tr><th v-for="(cell, index) in csvHeader" :key="`head-${index}`">{{ cell }}</th></tr></thead><tbody><tr v-for="(row, rowIndex) in csvBody" :key="`row-${rowIndex}`"><td v-for="(cell, cellIndex) in row" :key="`cell-${rowIndex}-${cellIndex}`">{{ cell }}</td></tr></tbody></table></div>
            </section>
            <section v-else-if="preview.kind === 'structured' && structuredTable && structuredViewMode === 'table'" class="csv-view">
              <div class="structured-toolbar"><span>{{ structuredTable.ruleName }}</span><span>{{ structuredTable.rows.length }} rows</span></div>
              <div class="csv-scroll"><table><thead><tr><th v-for="column in structuredTable.columns" :key="column">{{ column }}</th></tr></thead><tbody><tr v-for="(row, rowIndex) in structuredTable.rows" :key="`json-row-${rowIndex}`"><td v-for="(cell, cellIndex) in row" :key="`json-cell-${rowIndex}-${cellIndex}`">{{ cell }}</td></tr></tbody></table></div>
            </section>
            <pre v-else-if="preview.kind === 'structured'" class="structured-view"><code v-html="highlightCode(structuredText)" /></pre>
            <div v-else class="empty-preview"><strong>Preview unavailable</strong><span>{{ preview.content }}</span></div>
          </section>
        </div>
      </section>
    </section>
  </main>
  <SettingsDialog v-if="settingsVisible" v-model:settings="settingsDraft" v-model:rules="structuredRulesDraft" :error="settingsError" :archive-extensions="archiveExtensions" @close="settingsVisible = false" @save="saveSettings" />
</template>
