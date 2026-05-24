import { Archive, Download, FileAudio, FileText, FileVideo, Folder, FolderInput, Image, Info, LayoutGrid, Link2, List, MoreVertical, Pencil, RefreshCw, Star, Trash2 } from "lucide-react";
import React, { ChangeEvent, DragEvent, useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { createFolder, downloadFileUrl, DriveContents, DriveFile, DriveFolder, eventsUrl, listDriveContents, moveFile, moveFolder, renameFile, renameFolder, starFile, starFolder, thumbnailUrl, trashFile, trashFolder, zipFolderUrl } from "../api/agent";
import { useConfirm, useToast } from "../state/ui";
import { ContextMenu, ContextMenuItem } from "./ContextMenu";
import { DetailPanel } from "./DetailPanel";
import { MoveDialog } from "./MoveDialog";
import { ShareDialog } from "./ShareDialog";

type SortKey = "updated_at" | "name" | "size";
type ViewMode = "grid" | "list";
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
  const [menu, setMenu] = useState<{ x: number; y: number; items: ContextMenuItem[] } | null>(null);
  const [selection, setSelection] = useState<{ kind: "file"; data: DriveFile } | { kind: "folder"; data: DriveFolder } | null>(null);
  const [sortKey, setSortKey] = useState<SortKey>("updated_at");
  const [viewMode, setViewMode] = useState<ViewMode>("grid");
  const toast = useToast();
  const confirm = useConfirm();
  const folderInputRef = useRef<HTMLInputElement | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const currentFolderId = folderStack.length > 0 ? folderStack[folderStack.length - 1].id : "";

  const refresh = useCallback(async (folderId = currentFolderId) => {
    setLoading(true);
    setError(null);
    try {
      setContents(await listDriveContents(folderId));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [currentFolderId]);

  useEffect(() => { refresh(""); }, []);

  useEffect(() => {
    const stream = new EventSource(eventsUrl());
    stream.addEventListener("file.created", () => refresh(currentFolderId));
    stream.addEventListener("transfer.updated", () => refresh(currentFolderId));
    return () => stream.close();
  }, [currentFolderId, refresh]);

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
    await uploadQueue.enqueue(selected, { folderId: currentFolderId, preserveRelativePath: true });
  }

  function handleDragOver(event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    if (!dropping) setDropping(true);
  }

  function handleDragLeave(event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    if (event.currentTarget === event.target) setDropping(false);
  }

  async function handleDropOnFolder(folderId: string, event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    event.stopPropagation();
    setDropTargetFolder(null);
    if (event.dataTransfer.files && event.dataTransfer.files.length > 0) {
      const items = event.dataTransfer.items;
      const collected: { file: File; relativePath: string }[] = [];
      if (items && items.length > 0 && typeof items[0].webkitGetAsEntry === "function") {
        const entries = Array.from(items).map((item) => item.webkitGetAsEntry()).filter(Boolean) as FileSystemEntry[];
        for (const entry of entries) await collectEntry(entry, "", collected);
      } else {
        const fileList = Array.from(event.dataTransfer.files || []);
        for (const file of fileList) collected.push({ file, relativePath: file.name });
      }
      if (collected.length === 0) return;
      const usePath = collected.some((item) => item.relativePath.includes("/"));
      if (usePath) {
        const filesWithPath = collected.map((item) => attachRelativePath(item.file, item.relativePath));
        await uploadQueue.enqueue(filesWithPath, { folderId, preserveRelativePath: true });
      } else {
        await uploadQueue.enqueue(collected.map((item) => item.file), { folderId, preserveRelativePath: false });
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
    const items = event.dataTransfer.items;
    const collected: { file: File; relativePath: string }[] = [];
    if (items && items.length > 0 && typeof items[0].webkitGetAsEntry === "function") {
      const entries = Array.from(items).map((item) => item.webkitGetAsEntry()).filter(Boolean) as FileSystemEntry[];
      for (const entry of entries) {
        await collectEntry(entry, "", collected);
      }
    } else {
      const files = Array.from(event.dataTransfer.files || []);
      for (const file of files) collected.push({ file, relativePath: file.name });
    }
    if (collected.length === 0) return;
    const usePath = collected.some((item) => item.relativePath.includes("/"));
    if (usePath) {
      const filesWithPath = collected.map((item) => attachRelativePath(item.file, item.relativePath));
      await uploadQueue.enqueue(filesWithPath, { folderId: currentFolderId, preserveRelativePath: true });
    } else {
      await uploadQueue.enqueue(collected.map((item) => item.file), { folderId: currentFolderId, preserveRelativePath: false });
    }
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
  const sortedFiles = [...contents.files].sort(fileComparator(sortKey));

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
          {description && <p>{description}</p>}
          <div className="breadcrumb">
            <button onClick={goRoot}>{t("files.root")}</button>
            {folderStack.map((folder, index) => <button key={folder.id} onClick={() => goBreadcrumb(index)}>/{folder.name}</button>)}
          </div>
        </div>
        <div className="drive-browser__actions">
          <select className="select-control" value={sortKey} onChange={(event) => setSortKey(event.target.value as SortKey)}>
            <option value="updated_at">Mới nhất</option>
            <option value="name">Tên A-Z</option>
            <option value="size">Dung lượng</option>
          </select>
          <button className={`icon-button ${viewMode === "grid" ? "icon-button--active" : ""}`} onClick={() => setViewMode("grid")} aria-label="Lưới"><LayoutGrid size={16} /></button>
          <button className={`icon-button ${viewMode === "list" ? "icon-button--active" : ""}`} onClick={() => setViewMode("list")} aria-label="Danh sách"><List size={16} /></button>
          <button className="button button--primary" onClick={() => fileInputRef.current?.click()}>{t("files.upload")}</button>
          <button className="button button--secondary" onClick={() => folderInputRef.current?.click()}>{t("files.uploadFolder")}</button>
          <button className="button button--secondary" onClick={createNewFolder}>{t("files.createFolder")}</button>
          <button className="button button--ghost" onClick={() => refresh()} disabled={loading}><RefreshCw size={15} /></button>
          <input ref={fileInputRef} className="visually-hidden" type="file" multiple onChange={handleFileInput} />
          <input ref={folderInputRef} className="visually-hidden" type="file" multiple onChange={handleFolderInput} {...({ webkitdirectory: "", directory: "" } as Record<string, string>)} />
        </div>
      </div>
      {error && <div className="error-note">{error}</div>}
      {dropping && <div className="drop-overlay">{t("drive.dropHere")}</div>}
      {loading && <div className="muted-box">{t("files.loading")}</div>}
      {!loading && isEmpty && <div className="muted-box drop-hint">{t("drive.emptyDrop")}</div>}
      {!loading && !isEmpty && (
        <div className={viewMode === "grid" ? "file-grid" : "file-list"}>
          {sortedFolders.map((folder) => (
            <div
              className={`drive-card drive-card--folder ${dropTargetFolder === folder.id ? "drive-card--drop" : ""}`}
              key={folder.id}
              draggable
              onDragStart={(event) => { setDragItem({ kind: "folder", id: folder.id, name: folder.name }); event.dataTransfer.setData("text/plain", folder.name); }}
              onDragOver={(event) => { event.preventDefault(); setDropTargetFolder(folder.id); }}
              onDragLeave={() => setDropTargetFolder((current) => (current === folder.id ? null : current))}
              onDrop={(event) => handleDropOnFolder(folder.id, event)}
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
              className="drive-card"
              key={file.id}
              onClick={() => setSelection({ kind: "file", data: file })}
              draggable
              onDragStart={(event) => { setDragItem({ kind: "file", id: file.id, name: file.name }); event.dataTransfer.setData("text/plain", file.name); }}
              onDragEnd={() => setDragItem(null)}
            >
              <div className="drive-card__thumb">
                {file.preview_status === "ready" && file.kind === "image" ? <img src={thumbnailUrl(file.id)} alt="" /> : kindIcon(file.kind)}
              </div>
              <div className="drive-card__name">
                <strong>{file.name}{file.starred && <Star size={12} className="star-mark" />}</strong>
                <span>{kindLabel(file.kind)} · {formatBytes(file.size)}</span>
              </div>
              <div className="drive-card__footer">
                <span className={`badge badge--${syncBadge(file.sync_state)}`}>{syncLabel(file.sync_state)}</span>
                <button className="drive-card__action" onClick={(event) => { event.stopPropagation(); setShareTarget({ kind: "file", id: file.id, name: file.name }); }}><Link2 size={14} /></button>
                <a className="drive-card__action" href={downloadFileUrl(file.id)} onClick={(event) => event.stopPropagation()}><Download size={14} /></a>
                <button className="drive-card__menu" onClick={(event) => handleFileMenu(event, file)}><MoreVertical size={16} /></button>
              </div>
            </div>
          ))}
        </div>
      )}
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

async function collectEntry(entry: FileSystemEntry, parentPath: string, collected: { file: File; relativePath: string }[]) {
  if (entry.isFile) {
    await new Promise<void>((resolve, reject) => {
      (entry as FileSystemFileEntry).file((file) => {
        const relative = parentPath ? `${parentPath}/${file.name}` : file.name;
        collected.push({ file, relativePath: relative });
        resolve();
      }, reject);
    });
    return;
  }
  if (entry.isDirectory) {
    const reader = (entry as FileSystemDirectoryEntry).createReader();
    const children: FileSystemEntry[] = await new Promise((resolve) => {
      const list: FileSystemEntry[] = [];
      const read = () => reader.readEntries((items) => {
        if (items.length === 0) { resolve(list); return; }
        list.push(...items);
        read();
      });
      read();
    });
    const dirPath = parentPath ? `${parentPath}/${entry.name}` : entry.name;
    for (const child of children) {
      await collectEntry(child, dirPath, collected);
    }
  }
}

function kindIcon(kind: DriveFile["kind"]) {
  if (kind === "image") return <Image size={32} />;
  if (kind === "video") return <FileVideo size={32} />;
  if (kind === "audio") return <FileAudio size={32} />;
  if (kind === "archive") return <Archive size={32} />;
  return <FileText size={32} />;
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
