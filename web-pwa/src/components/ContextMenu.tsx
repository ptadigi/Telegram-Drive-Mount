import { ReactNode, useEffect, useRef, useState } from "react";

export type ContextMenuItem = {
  key: string;
  label: string;
  icon?: ReactNode;
  danger?: boolean;
  onSelect: () => void;
};

type Props = {
  items: ContextMenuItem[];
  position: { x: number; y: number };
  onClose: () => void;
};

export function ContextMenu({ items, position, onClose }: Props) {
  const ref = useRef<HTMLDivElement | null>(null);
  const [pos, setPos] = useState(position);

  useEffect(() => {
    function onClick(event: MouseEvent) {
      if (!ref.current) return;
      if (!ref.current.contains(event.target as Node)) onClose();
    }
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape") onClose();
    }
    window.addEventListener("mousedown", onClick);
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("mousedown", onClick);
      window.removeEventListener("keydown", onKey);
    };
  }, [onClose]);

  // Keep the menu inside the viewport (right-click / long-press near an edge).
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const rect = el.getBoundingClientRect();
    const pad = 8;
    let x = position.x;
    let y = position.y;
    if (x + rect.width + pad > window.innerWidth) x = Math.max(pad, window.innerWidth - rect.width - pad);
    if (y + rect.height + pad > window.innerHeight) y = Math.max(pad, window.innerHeight - rect.height - pad);
    setPos({ x, y });
  }, [position.x, position.y]);

  const style: React.CSSProperties = {
    position: "fixed",
    top: pos.y,
    left: pos.x,
    zIndex: 90,
  };

  return (
    <div ref={ref} className="context-menu" style={style} role="menu" aria-label="Tùy chọn">
      {items.map((item) => (
        <button
          key={item.key}
          role="menuitem"
          className={`context-menu__item ${item.danger ? "context-menu__item--danger" : ""}`}
          onClick={() => { item.onSelect(); onClose(); }}
        >
          {item.icon && <span className="context-menu__icon">{item.icon}</span>}
          <span>{item.label}</span>
        </button>
      ))}
    </div>
  );
}
