import { ChevronRight, Folder } from "lucide-react";
import { useEffect, useState } from "react";
import { DriveFolder, listDriveContents, moveFile, moveFolder } from "../api/agent";
import { useToast } from "../state/ui";

type MoveTarget = { kind: "file" | "folder"; id: string; name: string };

type Props = {
  open: boolean;
  onClose: () => void;
  targets: MoveTarget[];
  onMoved: () => void;
};

type Crumb = { id: string; name: string };

export function MoveDialog({ open, onClose, targets, onMoved }: Props) {
  const [stack, setStack] = useState<Crumb[]>([]);
  const [folders, setFolders] = useState<DriveFolder[]>([]);
  const [loading, setLoading] = useState(false);
  const toast = useToast();
  const currentFolderId = stack.length > 0 ? stack[stack.length - 1].id : "";
  const folderTargetIds = new Set(targets.filter((t) => t.kind === "folder").map((t) => t.id));
  const title = targets.length === 1 ? targets[0].name : `${targets.length} mục`;

  useEffect(() => {
    if (!open) return;
    setStack([]);
    setFolders([]);
    void browse("");
  }, [open]);

  async function browse(folderId: string) {
    setLoading(true);
    try {
      const result = await listDriveContents(folderId);
      // Never let a folder be moved into itself.
      setFolders(result.folders.filter((folder) => !folderTargetIds.has(folder.id)));
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err), "error");
    } finally {
      setLoading(false);
    }
  }

  async function openFolder(folder: DriveFolder) {
    setStack((prev) => [...prev, { id: folder.id, name: folder.name }]);
    await browse(folder.id);
  }

  async function goRoot() {
    setStack([]);
    await browse("");
  }

  async function goCrumb(index: number) {
    const next = stack.slice(0, index + 1);
    setStack(next);
    await browse(next[next.length - 1]?.id || "");
  }

  async function moveHere() {
    if (targets.length === 0) return;
    setLoading(true);
    let ok = 0;
    let failed = 0;
    for (const target of targets) {
      try {
        if (target.kind === "file") await moveFile(target.id, currentFolderId);
        else await moveFolder(target.id, currentFolderId);
        ok += 1;
      } catch {
        failed += 1;
      }
    }
    setLoading(false);
    if (failed === 0) toast(targets.length === 1 ? "Đã di chuyển" : `Đã di chuyển ${ok} mục`, "success");
    else toast(`Di chuyển ${ok} mục, lỗi ${failed} mục`, failed === targets.length ? "error" : "info");
    onMoved();
    onClose();
  }

  if (!open || targets.length === 0) return null;
  return (
    <div className="modal" onClick={(event) => event.target === event.currentTarget && onClose()}>
      <div className="modal__panel">
        <header className="modal__header">
          <div>
            <h2>Di chuyển {title}</h2>
            <p>Chọn thư mục đích, sau đó bấm Di chuyển vào đây.</p>
          </div>
          <button className="icon-button" onClick={onClose}>×</button>
        </header>
        <div className="breadcrumb">
          <button onClick={goRoot}>Drive của tôi</button>
          {stack.map((crumb, index) => <button key={crumb.id} onClick={() => goCrumb(index)}>/{crumb.name}</button>)}
        </div>
        <div className="move-list">
          {loading && <div className="muted-box">Đang tải...</div>}
          {!loading && folders.length === 0 && <div className="muted-box">Thư mục này chưa có thư mục con.</div>}
          {folders.map((folder) => (
            <button className="move-item" key={folder.id} onClick={() => openFolder(folder)}>
              <span className="move-item__icon"><Folder size={18} /></span>
              <span className="move-item__name">{folder.name}</span>
              <ChevronRight size={16} />
            </button>
          ))}
        </div>
        <footer className="modal__footer">
          <button className="button button--ghost" onClick={onClose}>Hủy</button>
          <button className="button button--primary" onClick={moveHere} disabled={loading}>Di chuyển vào đây</button>
        </footer>
      </div>
    </div>
  );
}
