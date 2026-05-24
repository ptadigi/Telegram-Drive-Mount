import { ReactNode, createContext, useCallback, useContext, useMemo, useRef, useState } from "react";

type ToastTone = "info" | "success" | "error";
type ToastItem = { id: number; tone: ToastTone; message: string };
type ConfirmRequest = {
  id: number;
  title: string;
  message: string;
  tone: ToastTone;
  onConfirm: () => void;
  onCancel?: () => void;
};

type UIContextValue = {
  toast: (message: string, tone?: ToastTone) => void;
  confirm: (options: { title?: string; message: string; tone?: ToastTone }) => Promise<boolean>;
};

const UIContext = createContext<UIContextValue | null>(null);

let counter = 0;

export function UIProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [confirmRequest, setConfirmRequest] = useState<ConfirmRequest | null>(null);
  const timersRef = useRef(new Map<number, number>());

  const removeToast = useCallback((id: number) => {
    setToasts((items) => items.filter((item) => item.id !== id));
    const timer = timersRef.current.get(id);
    if (timer) {
      window.clearTimeout(timer);
      timersRef.current.delete(id);
    }
  }, []);

  const toast = useCallback((message: string, tone: ToastTone = "info") => {
    counter += 1;
    const id = counter;
    setToasts((items) => [{ id, tone, message }, ...items]);
    const timer = window.setTimeout(() => removeToast(id), 3500);
    timersRef.current.set(id, timer);
  }, [removeToast]);

  const confirm = useCallback((options: { title?: string; message: string; tone?: ToastTone }) => {
    return new Promise<boolean>((resolve) => {
      counter += 1;
      const id = counter;
      setConfirmRequest({
        id,
        title: options.title || "Xác nhận",
        message: options.message,
        tone: options.tone || "info",
        onConfirm: () => { setConfirmRequest(null); resolve(true); },
        onCancel: () => { setConfirmRequest(null); resolve(false); },
      });
    });
  }, []);

  const value = useMemo<UIContextValue>(() => ({ toast, confirm }), [toast, confirm]);

  return (
    <UIContext.Provider value={value}>
      {children}
      <div className="toast-stack" aria-live="polite">
        {toasts.map((item) => (
          <button key={item.id} className={`toast toast--${item.tone}`} onClick={() => removeToast(item.id)}>{item.message}</button>
        ))}
      </div>
      {confirmRequest && (
        <div className="modal" onClick={(event) => event.target === event.currentTarget && confirmRequest.onCancel?.()}>
          <div className={`modal__panel modal__panel--confirm modal__panel--${confirmRequest.tone}`}>
            <header className="modal__header">
              <strong>{confirmRequest.title}</strong>
            </header>
            <p>{confirmRequest.message}</p>
            <footer className="modal__footer">
              <button className="button button--ghost" onClick={() => confirmRequest.onCancel?.()}>Hủy</button>
              <button className={`button ${confirmRequest.tone === "error" ? "button--danger" : "button--primary"}`} onClick={confirmRequest.onConfirm}>Xác nhận</button>
            </footer>
          </div>
        </div>
      )}
    </UIContext.Provider>
  );
}

export function useToast() {
  const context = useContext(UIContext);
  if (!context) throw new Error("useToast cần được bao trong UIProvider");
  return context.toast;
}

export function useConfirm() {
  const context = useContext(UIContext);
  if (!context) throw new Error("useConfirm cần được bao trong UIProvider");
  return context.confirm;
}
