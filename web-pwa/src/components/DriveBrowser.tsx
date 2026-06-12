import { Archive, CheckCircle2, Download, FileAudio, FileIcon, FileText, FileVideo, Folder, FolderInput, FolderPlus, FolderUp, Image, Info, LayoutGrid, Link2, List, MoreVertical, Pencil, RefreshCw, Star, Trash2, Upload } from "../icons";
import React, { ChangeEvent, DragEvent, useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { createFolder, downloadBundle, downloadFileUrl, DriveContents, DriveFile, DriveFolder, eventsUrl, listDriveContents, moveFile, moveFolder, renameFile, renameFolder, starFile, starFolder, thumbnailUrl, trashFile, trashFolder, zipFolderUrl } from "../api/agent";
import { useConfirm, useToast } from "../state/ui";
import { useRevalidate } from "../state/revalidate";
import { ContextMenu, ContextMenuItem } from "./ContextMenu";
import { DetailPanel } from "./DetailPanel";
import { FileViewer } from "./FileViewer";
import { MoveDialog } from "./MoveDialog";
import { ShareDialog } from "./ShareDialog";

type SortKey = "updated_at" | "name" | "size";
type ViewMode = "grid" | "list";
type KindFilter = "all" | "image" | "video" | "audio" | "document" | "archive" | "other";
import { UploadQueue } from "../state/uploads";

type Props = {
  uploadQueue: UploadQueue;
  rootLabel: string;
  description?: string;
};

export function DriveBrowser({ uploadQueue, rootLabel, description }: Props) {
  const { t } = useTranslation();
  const [contents, setContents] = useState<DriveContents>({ folders: [], files: [] });
  const [folderStack, setFolderStack] = useState<DriveFolder[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dropping, setDropping] = useState(false);
  const [shareTarget, setShareTarget] = useState<{ kind: "file" | "folder"; id: string; name: string } | null>(null);
  const [moveTarget, setMoveTarget] = useState<{ kind: "file" | "folder"; id: string; name: string } | null>(null);
  const [dropTargetFolder, setDropTargetFolder] = useState<string | null>(null);
  const [dragItem, setDragItem] = useState<{ kind: "file" | "folder"; id: string; name: string } | null>(null);
  const [selectedIds, setSelectedIds] = useState<{ files: Set<string>; folders: Set<string> }>({ files: new Set(), folders: new Set() });
  const [viewerFile, setViewerFile] = useState<DriveFile | null>(null);
  const [menu, setMenu] = useState<{ x: number; y: number; items: ContextMenuItem[] } | null>(null);
  const [selection, setSelection] = useState<{ kind: "file"; data: DriveFile } | { kind: "folder"; data: DriveFolder } | null>(null);
  const [sortKey, setSortKey] = useState<SortKey>("updated_at");
  const [viewMode, setViewMode] = useState<ViewMode>("grid");
  const [kindFilter, setKindFilter] = useState<KindFilter>("all");
  const toast = useToast();
  const confirm = useConfirm();
  const folderInputRef = useRef<HTMLInputElement | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const currentFolderId = folderStack.length > 0 ? folderStack[folderStack.length - 1].id : "";

  useEffect(() => {
    function onQuickAction(event: Event) {
      const action = (event as CustomEvent<{ action?: string }>).detail?.action;
      if (action === "file") fileInputRef.current?.click();
      if (action === "folder") folderInputRef.current?.click();
      if (action === "newfolder") createNewFolder();
    }
    window.addEventListener("drive:quick-action", onQuickAction);
    return () => window.removeEventListener("drive:quick-action", onQuickAction);
  }, [currentFolderId]);

  // Global guard for file drags. The browser's default action for a dropped
  // file is to OPEN/DOWNLOAD it (replacing the page). To prevent that anywhere
  // on the document we must preventDefault() on BOTH dragover AND drop for any
  // drag carrying files — not just outside the dropzone. The React onDrop
  // handlers on .drive-browser / .drive-card--folder still run (they fire
  // earlier in the bubble phase) and perform the actual upload enqueue; this
  // window-level guard only stops the browser from hijacking the drop.
  useEffect(() => {
    function hasFiles(dt: DataTransfer | null): boolean {
      return !!dt && Array.from(dt.types).includes("Files");
    }
    function isInsideDropzone(target: EventTarget | null): boolean {
      return target instanceof Element && !!target.closest(".drive-browser, .drive-card--folder");
    }
    function onWindowDragOver(event: Event) {
      const e = event as globalThis.DragEvent;
      if (!hasFiles(e.dataTransfer)) return;
      // Always cancel the default so the browser never opens the file.
      e.preventDefault();
      if (e.dataTransfer) {
        e.dataTransfer.dropEffect = isInsideDropzone(e.target) ? "copy" : "none";
      }
    }
    function onWindowDrop(event: Event) {
      const e = event as globalThis.DragEvent;
      if (!hasFiles(e.dataTransfer)) return;
      // Cancel the browser default everywhere. Inside the dropzone the React
      // onDrop already enqueued the files; outside we simply swallow it.
      e.preventDefault();
      if (!isInsideDropzone(e.target)) setDropping(false);
    }
    window.addEventListener("dragover", onWindowDragOver);
    window.addEventListener("drop", onWindowDrop);
    return () => {
      window.removeEventListener("dragover", onWindowDragOver);
      window.removeEventListener("drop", onWindowDrop);
    };
  }, []);

  const refresh = useCallback(async (folderId = currentFolderId, opts?: { silent?: boolean }) => {
    // Background revalidations (poll/focus/SSE/upload progress) run silently so
    // the list never flashes the loading state. Only explicit navigation and
    // the first load show the spinner.
    const silent = opts?.silent ?? false;
    if (!silent) setLoading(true);
    setError(null);
    try {
      const next = await listDriveContents(folderId);
      setContents(next);
    } catch (err) {
      if (!silent) setError(err instanceof Error ? err.message : String(err));
    } finally {
      if (!silent) setLoading(false);
    }
  }, [currentFolderId]);

  useEffect(() => { refresh(""); }, []);

  // Stale-while-revalidate: SSE is the fast path, but focus/visibility/online
  // and a light poll guarantee the list refreshes even when SSE is dropped by
  // a proxy (Cloudflare) or the mobile tab was suspended.
  useRevalidate(() => refresh(currentFolderId, { silent: true }), {
    eventsUrl: eventsUrl(),
    sseEvents: ["file.created", "file.updated", "file.trashed", "folder.updated", "transfer.updated"],
    pollMs: 20000,
  });

  // While uploads are active or processing, keep revalidating the current
  // folder so files appear as soon as the backend finishes importing them.
  const uploadInFlight = uploadQueue.items.some((i) => i.phase === "queued" || i.phase === "uploading_agent" || i.phase === "processing");
  useEffect(() => {
    if (!uploadInFlight) return;
    refresh(currentFolderId, { silent: true });
    const timer = window.setInterval(() => refresh(currentFolderId, { silent: true }), 2000);
    return () => window.clearInterval(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [uploadInFlight, currentFolderId]);

  // Once the queue settles, do one final refresh to catch the last imported
  // file if the backend finished just after the last interval tick.
  const settledSyncedCount = uploadQueue.items.filter((i) => i.phase === "synced").length;
  useEffect(() => {
    if (settledSyncedCount > 0) refresh(currentFolderId, { silent: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [settledSyncedCount]);

  function openFolder(folder: DriveFolder) {
    setFolderStack((stack) => [...stack, folder]);
    refresh(folder.id);
  }

  function goRoot() {
    setFolderStack([]);
    refresh("");
  }

  function goBreadcrumb(index: number) {
    const next = folderStack.slice(0, index + 1);
    setFolderStack(next);
    refresh(next.length > 0 ? next[next.length - 1].id : "");
  }

  async function createNewFolder() {
    const name = window.prompt(t("files.folderNamePrompt"));
    if (!name) return;
    try {
      const result = await createFolder(name, currentFolderId);
      setContents(result.contents);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function handleFileInput(event: ChangeEvent<HTMLInputElement>) {
    const selected = Array.from(event.target.files || []);
    event.target.value = "";
    if (selected.length === 0) return;
    await uploadQueue.enqueue(selected, { folderId: currentFolderId, preserveRelativePath: false });
  }

  async function handleFolderInput(event: ChangeEvent<HTMLInputElement>) {
    const selected = Array.from(event.target.files || []);
    event.target.value = "";
    if (selected.length === 0) return;
    // Enqueue in batches so picking a folder with tens of thousands of files
    // doesn't build one giant queue update or block the UI in a single tick.
    const BATCH = 200;
    for (let i = 0; i < selected.length; i += BATCH) {
      await uploadQueue.enqueue(selected.slice(i, i + BATCH), { folderId: currentFolderId, preserveRelativePath: true });
      await new Promise((r) => setTimeout(r, 0));
    }
  }

  function clearSelection() {
    setSelectedIds({ files: new Set(), folders: new Set() });
  }

  function toggleSelectFile(id: string, additive: boolean) {
    setSelectedIds((current) => {
      const files = new Set(additive ? current.files : []);
      const folders = new Set(additive ? current.folders : []);
      if (files.has(id)) files.delete(id); else files.add(id);
      return { files, folders };
    });
  }

  function toggleSelectFolder(id: string, additive: boolean) {
    setSelectedIds((current) => {
      const files = new Set(additive ? current.files : []);
      const folders = new Set(additive ? current.folders : []);
      if (folders.has(id)) folders.delete(id); else folders.add(id);
      return { files, folders };
    });
  }

  async function bulkTrash() {
    const fileCount = selectedIds.files.size;
    const folderCount = selectedIds.folders.size;
    if (fileCount === 0 && folderCount === 0) return;
    const ok = await confirm({ title: "Xóa nhiều mục", message: `Đưa ${fileCount} file và ${folderCount} thư mục vào thùng rác?`, tone: "error" });
    if (!ok) return;
    try {
      for (const id of selectedIds.files) await trashFile(id);
      for (const id of selectedIds.folders) await trashFolder(id);
      clearSelection();
      toast(`Đã xóa ${fileCount + folderCount} mục`, "success");
      refresh();
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err), "error");
    }
  }

  async function bulkStar(value: boolean) {
    try {
      for (const id of selectedIds.files) await starFile(id, value);
      for (const id of selectedIds.folders) await starFolder(id, value);
      clearSelection();
      toast(value ? "Đã đánh dấu sao" : "Đã bỏ đánh dấu sao", "success");
      refresh();
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err), "error");
    }
  }

  function bulkMove() {
    const fileCount = selectedIds.files.size;
    const folderCount = selectedIds.folders.size;
    if (fileCount + folderCount === 0) return;
    const total = fileCount + folderCount;
    if (total === 1) {
      const fileId = selectedIds.files.values().next().value as string | undefined;
      const folderId = selectedIds.folders.values().next().value as string | undefined;
      if (fileId) {
        const file = contents.files.find((item) => item.id === fileId);
        if (file) setMoveTarget({ kind: "file", id: file.id, name: file.name });
      } else if (folderId) {
        const folder = contents.folders.find((item) => item.id === folderId);
        if (folder) setMoveTarget({ kind: "folder", id: folder.id, name: folder.name });
      }
      return;
    }
    toast("Hiện chỉ hỗ trợ di chuyển từng mục, hãy chọn 1 mục", "info");
  }

  function handleDragOver(event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    if (event.dataTransfer.types.includes("Files") && !dropping) setDropping(true);
  }

  function handleDragLeave(event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    if (event.currentTarget === event.target) setDropping(false);
  }

  async function handleDropOnFolder(folderId: string, event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    event.stopPropagation();
    setDropTargetFolder(null);
    setDropping(false);
    if (event.dataTransfer.files && event.dataTransfer.files.length > 0) {
      const items = event.dataTransfer.items;
      if (items && items.length > 0 && typeof items[0].webkitGetAsEntry === "function") {
        const entries = Array.from(items).map((item) => item.webkitGetAsEntry()).filter(Boolean) as FileSystemEntry[];
        const hasDir = entries.some((e) => e.isDirectory);
        if (hasDir) toast("Đang quét thư mục và tải lên…", "info");
        const total = await streamEntries(entries, async (batch) => {
          const usePath = batch.some((item) => item.relativePath.includes("/"));
          if (usePath) {
            const withPath = batch.map((item) => attachRelativePath(item.file, item.relativePath));
            await uploadQueue.enqueue(withPath, { folderId, preserveRelativePath: true });
          } else {
            await uploadQueue.enqueue(batch.map((item) => item.file), { folderId, preserveRelativePath: false });
          }
        });
        if (hasDir) toast(`Đã đưa ${total} file vào hàng đợi tải lên`, "success");
      } else {
        const fileList = Array.from(event.dataTransfer.files || []);
        if (fileList.length === 0) return;
        await uploadQueue.enqueue(fileList, { folderId, preserveRelativePath: false });
      }
      return;
    }
    if (!dragItem) return;
    if (dragItem.kind === "folder" && dragItem.id === folderId) return;
    try {
      if (dragItem.kind === "file") await moveFile(dragItem.id, folderId);
      else await moveFolder(dragItem.id, folderId);
      toast(`Đã chuyển ${dragItem.name}`, "success");
      refresh();
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err), "error");
    } finally {
      setDragItem(null);
    }
  }

  async function handleDropOnRoot(event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    event.stopPropagation();
    setDropTargetFolder(null);
    setDropping(false);
    if (!dragItem) {
      handleDrop(event);
      return;
    }
    try {
      if (dragItem.kind === "file") await moveFile(dragItem.id, "");
      else await moveFolder(dragItem.id, "");
      toast(`Đã chuyển ${dragItem.name} ra root`, "success");
      refresh();
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err), "error");
    } finally {
      setDragItem(null);
    }
  }

  async function handleDrop(event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    setDropping(false);
    setDropTargetFolder(null);
    const items = event.dataTransfer.items;
    if (items && items.length > 0 && typeof items[0].webkitGetAsEntry === "function") {
      const entries = Array.from(items).map((item) => item.webkitGetAsEntry()).filter(Boolean) as FileSystemEntry[];
      const hasDir = entries.some((e) => e.isDirectory);
      if (hasDir) toast("Đang quét thư mục và tải lên…", "info");
      // Stream batches straight into the upload queue so uploads start while
      // the (possibly huge) tree is still being scanned, with bounded memory.
      const total = await streamEntries(entries, async (batch) => {
        const usePath = batch.some((item) => item.relativePath.includes("/"));
        if (usePath) {
          const withPath = batch.map((item) => attachRelativePath(item.file, item.relativePath));
          await uploadQueue.enqueue(withPath, { folderId: currentFolderId, preserveRelativePath: true });
        } else {
          await uploadQueue.enqueue(batch.map((item) => item.file), { folderId: currentFolderId, preserveRelativePath: false });
        }
      });
      if (hasDir) toast(`Đã đưa ${total} file vào hàng đợi tải lên`, "success");
      return;
    }
    const files = Array.from(event.dataTransfer.files || []);
    if (files.length === 0) return;
    await uploadQueue.enqueue(files, { folderId: currentFolderId, preserveRelativePath: false });
  }

  function handleFolderMenu(event: React.MouseEvent, folder: DriveFolder) {
    event.preventDefault();
    event.stopPropagation();
    setSelection({ kind: "folder", data: folder });
    setMenu({
      x: event.clientX,
      y: event.clientY,
      items: [
        { key: "rename", label: t("files.rename"), icon: <Pencil size={14} />, onSelect: () => promptRenameFolder(folder) },
        { key: "move", label: "Di chuyển đến", icon: <FolderInput size={14} />, onSelect: () => setMoveTarget({ kind: "folder", id: folder.id, name: folder.name }) },
        { key: "share", label: t("files.share"), icon: <Link2 size={14} />, onSelect: () => setShareTarget({ kind: "folder", id: folder.id, name: folder.name }) },
        { key: "star", label: folder.starred ? "Bỏ đánh dấu sao" : "Đánh dấu sao", icon: <Star size={14} />, onSelect: () => toggleStarFolder(folder) },
        { key: "zip", label: "Tải xuống dạng ZIP", icon: <Download size={14} />, onSelect: () => window.location.assign(zipFolderUrl(folder.id)) },
        { key: "trash", label: t("files.trash"), icon: <Trash2 size={14} />, danger: true, onSelect: () => promptTrashFolder(folder) },
      ],
    });
  }

  function handleFileMenu(event: React.MouseEvent, file: DriveFile) {
    event.preventDefault();
    event.stopPropagation();
    setSelection({ kind: "file", data: file });
    setMenu({
      x: event.clientX,
      y: event.clientY,
      items: [
        { key: "details", label: t("files.viewDetails"), icon: <Info size={14} />, onSelect: () => setSelection({ kind: "file", data: file }) },
        { key: "view", label: "Mở xem trực tiếp", icon: <FileText size={14} />, onSelect: () => setViewerFile(file) },
        { key: "share", label: t("files.share"), icon: <Link2 size={14} />, onSelect: () => setShareTarget({ kind: "file", id: file.id, name: file.name }) },
        { key: "star", label: file.starred ? "Bỏ đánh dấu sao" : "Đánh dấu sao", icon: <Star size={14} />, onSelect: () => toggleStarFile(file) },
        { key: "rename", label: t("files.rename"), icon: <Pencil size={14} />, onSelect: () => promptRenameFile(file) },
        { key: "move", label: "Di chuyển đến", icon: <FolderInput size={14} />, onSelect: () => setMoveTarget({ kind: "file", id: file.id, name: file.name }) },
        { key: "download", label: t("files.download"), icon: <Download size={14} />, onSelect: () => window.location.assign(downloadFileUrl(file.id)) },
        { key: "trash", label: t("files.trash"), icon: <Trash2 size={14} />, danger: true, onSelect: () => promptTrashFile(file) },
      ],
    });
  }

  function promptRenameFolder(folder: DriveFolder) {
    const name = window.prompt(t("files.renamePrompt"), folder.name);
    if (name && name !== folder.name) renameFolder(folder.id, name).then(() => { refresh(); toast(`Đã đổi tên thành ${name}`, "success"); }).catch((err) => toast(err instanceof Error ? err.message : String(err), "error"));
  }

  function promptRenameFile(file: DriveFile) {
    const name = window.prompt(t("files.renamePrompt"), file.name);
    if (name && name !== file.name) renameFile(file.id, name).then(() => { refresh(); toast(`Đã đổi tên thành ${name}`, "success"); }).catch((err) => toast(err instanceof Error ? err.message : String(err), "error"));
  }

  async function promptTrashFolder(folder: DriveFolder) {
    const ok = await confirm({ title: "Xóa thư mục", message: `Đưa ${folder.name} vào thùng rác?`, tone: "error" });
    if (!ok) return;
    try {
      await trashFolder(folder.id);
      await refresh();
      toast("Đã đưa thư mục vào thùng rác", "success");
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err), "error");
    }
  }

  async function promptTrashFile(file: DriveFile) {
    const ok = await confirm({ title: "Xóa file", message: `Đưa ${file.name} vào thùng rác?`, tone: "error" });
    if (!ok) return;
    try {
      await trashFile(file.id);
      await refresh();
      toast("Đã đưa file vào thùng rác", "success");
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err), "error");
    }
  }

  async function toggleStarFile(file: DriveFile) {
    try {
      const next = !file.starred;
      await starFile(file.id, next);
      toast(next ? "Đã đánh dấu sao" : "Đã bỏ đánh dấu sao", "success");
      refresh();
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err), "error");
    }
  }

  async function toggleStarFolder(folder: DriveFolder) {
    try {
      const next = !folder.starred;
      await starFolder(folder.id, next);
      toast(next ? "Đã đánh dấu sao" : "Đã bỏ đánh dấu sao", "success");
      refresh();
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err), "error");
    }
  }

  const isEmpty = contents.folders.length === 0 && contents.files.length === 0;
  const sortedFolders = [...contents.folders].sort(folderComparator(sortKey));
  const sortedFiles = [...contents.files].filter((file) => kindFilter === "all" || file.kind === kindFilter).sort(fileComparator(sortKey));

  return (
    <div className={selection ? "drive-layout drive-layout--with-detail" : "drive-layout"}>
      <section
        className={dropping ? "drive-browser drive-browser--dropping" : "drive-browser"}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDropOnRoot}
      >
      <div className="drive-browser__header">
        <div>
          <h2>{rootLabel}</h2>
          <div className="breadcrumb">
            <button onClick={goRoot}>{t("files.root")}</button>
            {folderStack.map((folder, index) => <button key={folder.id} onClick={() => goBreadcrumb(index)}>/{folder.name}</button>)}
          </div>
        </div>
        <div className="drive-browser__actions">
          <select className="select-control" value={sortKey} onChange={(event) => setSortKey(event.target.value as SortKey)} aria-label="Sắp xếp">
            <option value="updated_at">Mới nhất</option>
            <option value="name">Tên A-Z</option>
            <option value="size">Dung lượng</option>
          </select>
          <select className="select-control" value={kindFilter} onChange={(event) => setKindFilter(event.target.value as KindFilter)} aria-label="Lọc loại">
            <option value="all">Tất cả loại</option>
            <option value="image">Hình ảnh</option>
            <option value="video">Video</option>
            <option value="audio">Âm thanh</option>
            <option value="document">Tài liệu</option>
            <option value="archive">File nén</option>
            <option value="other">Khác</option>
          </select>
          <div className="view-toggle" role="group" aria-label="Chế độ hiển thị">
            <button className={`icon-button ${viewMode === "grid" ? "icon-button--active" : ""}`} onClick={() => setViewMode("grid")} aria-label="Lưới" title="Lưới"><LayoutGrid size={16} /></button>
            <button className={`icon-button ${viewMode === "list" ? "icon-button--active" : ""}`} onClick={() => setViewMode("list")} aria-label="Danh sách" title="Danh sách"><List size={16} /></button>
          </div>
          <button className="icon-button" onClick={() => refresh()} disabled={loading} aria-label="Làm mới" title="Làm mới"><RefreshCw size={15} /></button>
          <button className="icon-button" onClick={createNewFolder} aria-label={t("files.createFolder")} title={t("files.createFolder")}><FolderPlus size={16} /></button>
          <button className="icon-button" onClick={() => folderInputRef.current?.click()} aria-label={t("files.uploadFolder")} title={t("files.uploadFolder")}><FolderUp size={16} /></button>
          <button className="button button--primary" onClick={() => fileInputRef.current?.click()}><Upload size={15} /> {t("files.upload")}</button>
          <input ref={fileInputRef} className="visually-hidden" type="file" multiple onChange={handleFileInput} />
          <input ref={folderInputRef} className="visually-hidden" type="file" multiple onChange={handleFolderInput} {...({ webkitdirectory: "", directory: "" } as Record<string, string>)} />
        </div>
      </div>
      {selectedIds.files.size + selectedIds.folders.size > 0 && (
        <div className="bulk-bar">
          <span>Đã chọn {selectedIds.files.size + selectedIds.folders.size} mục</span>
          <div className="bulk-bar__actions">
            <button className="button button--ghost" onClick={() => bulkStar(true)}>★ Sao</button>
            <button className="button button--ghost" onClick={() => bulkStar(false)}>Bỏ sao</button>
            <button className="button button--ghost" onClick={bulkMove}>Di chuyển</button>
            <button className="button button--ghost" onClick={async () => {
              try {
                await downloadBundle([...selectedIds.files], [...selectedIds.folders]);
                clearSelection();
                toast("Đã chuẩn bị file ZIP", "success");
              } catch (err) {
                toast(err instanceof Error ? err.message : String(err), "error");
              }
            }}>Tải ZIP</button>
            <button className="button button--danger" onClick={bulkTrash}>Xóa vào thùng rác</button>
            <button className="button button--ghost" onClick={clearSelection}>Hủy</button>
          </div>
        </div>
      )}
      {error && <div className="error-note">{error}</div>}
      {dropping && <div className="drop-overlay">{t("drive.dropHere")}</div>}
      {loading && <div className="muted-box">{t("files.loading")}</div>}
      {!loading && isEmpty && <div className="muted-box drop-hint">{t("drive.emptyDrop")}</div>}
      {!loading && !isEmpty && (
        <div className={viewMode === "grid" ? "file-grid" : "file-list"}>
          {sortedFolders.map((folder) => (
            <div
              className={`drive-card drive-card--folder ${dropTargetFolder === folder.id ? "drive-card--drop" : ""} ${selectedIds.folders.has(folder.id) ? "drive-card--selected" : ""}`}
              key={folder.id}
              draggable
              onDragStart={(event) => { setDragItem({ kind: "folder", id: folder.id, name: folder.name }); event.dataTransfer.setData("text/plain", folder.name); }}
              onDragOver={(event) => { event.preventDefault(); setDropTargetFolder(folder.id); }}
              onDragLeave={() => setDropTargetFolder((current) => (current === folder.id ? null : current))}
              onDrop={(event) => handleDropOnFolder(folder.id, event)}
              onClick={(event) => {
                if (event.shiftKey || event.ctrlKey || event.metaKey) {
                  toggleSelectFolder(folder.id, event.shiftKey || event.ctrlKey || event.metaKey);
                  return;
                }
                if (selectedIds.files.size + selectedIds.folders.size > 0) {
                  toggleSelectFolder(folder.id, true);
                }
              }}
              onDoubleClick={() => openFolder(folder)}
            >
              <button className="drive-card__open" onClick={() => openFolder(folder)}>
                <div className="drive-card__thumb"><Folder size={36} /></div>
                <div className="drive-card__name"><strong>{folder.name}{folder.starred && <Star size={12} className="star-mark" />}</strong><span>{t("files.folder")}</span></div>
              </button>
              <button className="drive-card__menu" onClick={(event) => handleFolderMenu(event, folder)}><MoreVertical size={16} /></button>
            </div>
          ))}
          {sortedFiles.map((file) => (
            <div
              className={`drive-card ${selectedIds.files.has(file.id) ? "drive-card--selected" : ""}`}
              key={file.id}
              onClick={(event) => {
                if (event.shiftKey || event.ctrlKey || event.metaKey) {
                  toggleSelectFile(file.id, event.shiftKey || event.ctrlKey || event.metaKey);
                  return;
                }
                if (selectedIds.files.size + selectedIds.folders.size > 0) {
                  toggleSelectFile(file.id, true);
                  return;
                }
                setSelection({ kind: "file", data: file });
              }}
              onDoubleClick={() => setViewerFile(file)}
              draggable
              onDragStart={(event) => { setDragItem({ kind: "file", id: file.id, name: file.name }); event.dataTransfer.setData("text/plain", file.name); }}
              onDragEnd={() => setDragItem(null)}
            >
              <div className="drive-card__thumb">
                <CardThumb file={file} />
              </div>
              <div className="drive-card__name">
                <strong>{file.name}{file.starred && <Star size={12} className="star-mark" />}</strong>
                <span>{kindLabel(file.kind)} · {formatBytes(file.size)}</span>
              </div>
              <div className="drive-card__footer">
                <span className={`badge badge--${syncBadge(file.sync_state)}`} title={syncLabel(file.sync_state)}>
                  {syncBadge(file.sync_state) === "ok" ? <CheckCircle2 size={14} /> : syncLabel(file.sync_state)}
                </span>
                <button className="drive-card__action" onClick={(event) => { event.stopPropagation(); setShareTarget({ kind: "file", id: file.id, name: file.name }); }}><Link2 size={14} /></button>
                <a className="drive-card__action" href={downloadFileUrl(file.id)} onClick={(event) => event.stopPropagation()}><Download size={14} /></a>
                <button className="drive-card__menu" onClick={(event) => handleFileMenu(event, file)}><MoreVertical size={16} /></button>
              </div>
            </div>
          ))}
        </div>
      )}
      <FileViewer file={viewerFile} onClose={() => setViewerFile(null)} />
      <ShareDialog
        open={!!shareTarget}
        onClose={() => setShareTarget(null)}
        targetKind={shareTarget?.kind || "file"}
        targetId={shareTarget?.id || ""}
        targetName={shareTarget?.name || ""}
      />
      <MoveDialog
        open={!!moveTarget}
        onClose={() => setMoveTarget(null)}
        target={moveTarget}
        onMoved={refresh}
      />
      </section>
      <DetailPanel selection={selection} onClose={() => setSelection(null)} onShare={() => {
        if (!selection) return;
        if (selection.kind === "file") setShareTarget({ kind: "file", id: selection.data.id, name: selection.data.name });
        else setShareTarget({ kind: "folder", id: selection.data.id, name: selection.data.name });
      }} />
      {menu && <ContextMenu position={{ x: menu.x, y: menu.y }} items={menu.items} onClose={() => setMenu(null)} />}
    </div>
  );
}

function attachRelativePath(file: File, relativePath: string) {
  Object.defineProperty(file, "webkitRelativePath", { value: relativePath, configurable: true });
  return file;
}

function folderComparator(sortKey: SortKey) {
  return (a: DriveFolder, b: DriveFolder) => {
    if (sortKey === "name") return a.name.localeCompare(b.name, "vi");
    return b.updated_at - a.updated_at;
  };
}

function fileComparator(sortKey: SortKey) {
  return (a: DriveFile, b: DriveFile) => {
    if (sortKey === "name") return a.name.localeCompare(b.name, "vi");
    if (sortKey === "size") return b.size - a.size;
    return b.updated_at - a.updated_at;
  };
}

// streamEntries walks dropped FileSystem entries and flushes files to onBatch
// in chunks instead of collecting everything into one array first. This keeps
// memory bounded and lets uploads start while the tree is still being scanned —
// essential for folders with tens of thousands of files (the old "collect all
// then enqueue" approach held every File in memory and blocked the UI). It also
// yields to the event loop between batches so the tab stays responsive.
async function streamEntries(
  entries: FileSystemEntry[],
  onBatch: (batch: { file: File; relativePath: string }[]) => Promise<void> | void,
  batchSize = 200,
): Promise<number> {
  let total = 0;
  let buffer: { file: File; relativePath: string }[] = [];

  const flush = async () => {
    if (buffer.length === 0) return;
    const batch = buffer;
    buffer = [];
    await onBatch(batch);
    // Yield so the browser can paint/respond between batches.
    await new Promise((r) => setTimeout(r, 0));
  };

  // Iterative DFS with an explicit stack to avoid deep recursion on large trees.
  const stack: { entry: FileSystemEntry; parentPath: string }[] = entries.map((entry) => ({ entry, parentPath: "" }));
  while (stack.length > 0) {
    const { entry, parentPath } = stack.pop()!;
    if (entry.isFile) {
      const file = await new Promise<File>((resolve, reject) => {
        (entry as FileSystemFileEntry).file(resolve, reject);
      });
      const relative = parentPath ? `${parentPath}/${file.name}` : file.name;
      buffer.push({ file, relativePath: relative });
      total += 1;
      if (buffer.length >= batchSize) await flush();
    } else if (entry.isDirectory) {
      const reader = (entry as FileSystemDirectoryEntry).createReader();
      const dirPath = parentPath ? `${parentPath}/${entry.name}` : entry.name;
      // readEntries returns at most ~100 entries per call; loop until empty.
      // Push children onto the stack so we never recurse and never hold the
      // whole tree in memory at once.
      // eslint-disable-next-line no-constant-condition
      while (true) {
        const items: FileSystemEntry[] = await new Promise((resolve, reject) => {
          reader.readEntries((res) => resolve(res), reject);
        });
        if (items.length === 0) break;
        for (const child of items) stack.push({ entry: child, parentPath: dirPath });
      }
    }
  }
  await flush();
  return total;
}

function canHaveThumb(file: DriveFile): boolean {
  const ext = (file.extension || "").toLowerCase();
  return file.kind === "image" || file.kind === "video" || (file.kind === "document" && ext === ".pdf");
}

// CardThumb shows a bitmap thumbnail for images/videos/PDFs. It requests the
// thumbnail even when preview_status isn't "ready" yet, which triggers the
// agent's lazy (re)generation for files uploaded by older builds. On any error
// (no thumbnail, missing ffmpeg/poppler) it falls back to the kind icon.
function CardThumb({ file }: { file: DriveFile }) {
  const [failed, setFailed] = useState(false);
  if (!canHaveThumb(file) || failed) return <FileIcon kind={file.kind} ext={file.extension} size={32} />;
  return <img src={thumbnailUrl(file.id)} alt="" loading="lazy" onError={() => setFailed(true)} />;
}

function kindLabel(kind: string) {
  const labels: Record<string, string> = { image: "Hình ảnh", video: "Video", audio: "Âm thanh", document: "Tài liệu", archive: "Nén", other: "File" };
  return labels[kind] || "File";
}

function syncLabel(state: string) {
  const labels: Record<string, string> = { pending_telegram_upload: "Chờ đồng bộ", telegram_uploading: "Đang đồng bộ", telegram_synced: "Đã đồng bộ", telegram_upload_failed: "Lỗi đồng bộ", metadata_only: "Metadata" };
  return labels[state] || state;
}

function syncBadge(state: string) {
  if (state === "telegram_synced") return "ok";
  if (state === "telegram_upload_failed") return "error";
  if (state === "telegram_uploading") return "active";
  return "pending";
}

function formatBytes(bytes: number) {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) { value /= 1024; unit += 1; }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unit]}`;
}
