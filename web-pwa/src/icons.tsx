// Central icon module. The whole PWA imports icons from here instead of
// directly from an icon library, so we have ONE consistent icon set
// (Solar, linear/stroke style — matches our clean SaaS aesthetic) and a single
// place to swap sets later.
//
// Icons are compiled to inline SVG at build time by unplugin-icons (no runtime
// network fetch — safe for an offline/self-hosted PWA). Each Solar glyph is
// imported as a component, then wrapped so call sites keep the lucide-style API
// (`size`, `className`, `color`, `strokeWidth` is ignored — Solar linear has a
// fixed stroke).
import type { ComponentType, SVGProps } from "react";

import SolarArchive from "~icons/solar/archive-linear";
import SolarCalendar from "~icons/solar/calendar-linear";
import SolarCheckCircle from "~icons/solar/check-circle-linear";
import SolarArrowDown from "~icons/solar/alt-arrow-down-linear";
import SolarArrowRight from "~icons/solar/alt-arrow-right-linear";
import SolarArrowUp from "~icons/solar/alt-arrow-up-linear";
import SolarClock from "~icons/solar/clock-circle-linear";
import SolarCloud from "~icons/solar/cloud-linear";
import SolarCopy from "~icons/solar/copy-linear";
import SolarDatabase from "~icons/solar/database-linear";
import SolarDownload from "~icons/solar/download-minimalistic-linear";
import SolarExternal from "~icons/solar/square-arrow-right-up-linear";
import SolarMusic from "~icons/solar/music-note-2-linear";
import SolarFileText from "~icons/solar/file-text-linear";
import SolarVideo from "~icons/solar/videocamera-linear";
import SolarFolder from "~icons/solar/folder-linear";
import SolarFolderFiles from "~icons/solar/folder-with-files-linear";
import SolarFolderOpen from "~icons/solar/folder-open-linear";
import SolarAddFolder from "~icons/solar/add-folder-linear";
import SolarFolderCloud from "~icons/solar/folder-cloud-linear";
import SolarGlobal from "~icons/solar/global-linear";
import SolarSsd from "~icons/solar/ssd-square-linear";
import SolarHome from "~icons/solar/home-2-linear";
import SolarGallery from "~icons/solar/gallery-linear";
import SolarInfo from "~icons/solar/info-circle-linear";
import SolarKey from "~icons/solar/key-linear";
import SolarLaptop from "~icons/solar/laptop-2-linear";
import SolarWidget from "~icons/solar/widget-4-linear";
import SolarLink from "~icons/solar/link-linear";
import SolarList from "~icons/solar/list-linear";
import SolarRefresh from "~icons/solar/refresh-linear";
import SolarLock from "~icons/solar/lock-linear";
import SolarMenuDots from "~icons/solar/menu-dots-linear";
import SolarPause from "~icons/solar/pause-linear";
import SolarPen from "~icons/solar/pen-2-linear";
import SolarPhone from "~icons/solar/phone-linear";
import SolarPlay from "~icons/solar/play-linear";
import SolarAddCircle from "~icons/solar/add-circle-linear";
import SolarQr from "~icons/solar/qr-code-linear";
import SolarRestart from "~icons/solar/restart-linear";
import SolarRefreshCircle from "~icons/solar/refresh-circle-linear";
import SolarMagnifer from "~icons/solar/magnifer-linear";
import SolarServer from "~icons/solar/server-2-linear";
import SolarSettings from "~icons/solar/settings-linear";
import SolarShare from "~icons/solar/share-linear";
import SolarShieldCheck from "~icons/solar/shield-check-linear";
import SolarShieldCross from "~icons/solar/shield-cross-linear";
import SolarMagicStick from "~icons/solar/magic-stick-3-linear";
import SolarStar from "~icons/solar/star-linear";
import SolarTrash from "~icons/solar/trash-bin-trash-linear";
import SolarUpload from "~icons/solar/upload-minimalistic-linear";
import SolarCloudUpload from "~icons/solar/cloud-upload-linear";
import SolarWifi from "~icons/solar/wi-fi-router-linear";
import SolarClose from "~icons/solar/close-circle-linear";
import SolarZoomIn from "~icons/solar/magnifer-zoom-in-linear";
import SolarZoomOut from "~icons/solar/magnifer-zoom-out-linear";

export type IconProps = SVGProps<SVGSVGElement> & {
  size?: number | string;
  // Accepted for lucide API compatibility; Solar linear has a fixed stroke.
  strokeWidth?: number | string;
};

type SvgComp = ComponentType<SVGProps<SVGSVGElement>>;

function wrap(Comp: SvgComp) {
  return function Icon({ size = 24, strokeWidth: _sw, width, height, ...rest }: IconProps) {
    return <Comp width={width ?? size} height={height ?? size} {...rest} />;
  };
}

// Public icon set — names match the previous lucide-react imports so call sites
// don't change beyond their import path.
export const Archive = wrap(SolarArchive);
export const CalendarClock = wrap(SolarCalendar);
export const CheckCircle2 = wrap(SolarCheckCircle);
export const ChevronDown = wrap(SolarArrowDown);
export const ChevronRight = wrap(SolarArrowRight);
export const ChevronUp = wrap(SolarArrowUp);
export const Clock3 = wrap(SolarClock);
export const Cloud = wrap(SolarCloud);
export const Copy = wrap(SolarCopy);
export const Database = wrap(SolarDatabase);
export const Download = wrap(SolarDownload);
export const ExternalLink = wrap(SolarExternal);
export const FileAudio = wrap(SolarMusic);
export const FileText = wrap(SolarFileText);
export const FileVideo = wrap(SolarVideo);
export const Folder = wrap(SolarFolder);
export const FolderInput = wrap(SolarFolderFiles);
export const FolderOpen = wrap(SolarFolderOpen);
export const FolderPlus = wrap(SolarAddFolder);
export const FolderSync = wrap(SolarFolderCloud);
export const FolderUp = wrap(SolarFolderFiles);
export const Globe = wrap(SolarGlobal);
export const HardDrive = wrap(SolarSsd);
export const Home = wrap(SolarHome);
export const Image = wrap(SolarGallery);
export const Info = wrap(SolarInfo);
export const KeyRound = wrap(SolarKey);
export const Laptop2 = wrap(SolarLaptop);
export const LayoutGrid = wrap(SolarWidget);
export const Link2 = wrap(SolarLink);
export const List = wrap(SolarList);
export const Loader2 = wrap(SolarRefresh);
export const Lock = wrap(SolarLock);
export const MoreVertical = wrap(SolarMenuDots);
export const Pause = wrap(SolarPause);
export const Pencil = wrap(SolarPen);
export const Phone = wrap(SolarPhone);
export const Play = wrap(SolarPlay);
export const Plus = wrap(SolarAddCircle);
export const QrCode = wrap(SolarQr);
export const RefreshCw = wrap(SolarRefresh);
export const RotateCcw = wrap(SolarRestart);
export const RotateCw = wrap(SolarRefreshCircle);
export const Search = wrap(SolarMagnifer);
export const Server = wrap(SolarServer);
export const Settings = wrap(SolarSettings);
export const Share2 = wrap(SolarShare);
export const ShieldCheck = wrap(SolarShieldCheck);
export const ShieldOff = wrap(SolarShieldCross);
export const Sparkles = wrap(SolarMagicStick);
export const Star = wrap(SolarStar);
export const Trash2 = wrap(SolarTrash);
export const Upload = wrap(SolarUpload);
export const UploadCloud = wrap(SolarCloudUpload);
export const Wifi = wrap(SolarWifi);
export const X = wrap(SolarClose);
export const XCircle = wrap(SolarClose);
export const ZoomIn = wrap(SolarZoomIn);
export const ZoomOut = wrap(SolarZoomOut);

// ===================== Coloured file-type icons =====================
// Google-Drive-style: each format gets a distinct duotone glyph + brand-ish
// colour so users recognise file types at a glance.
import SolarDocText from "~icons/solar/document-text-bold-duotone";
import SolarDoc from "~icons/solar/document-bold-duotone";
import SolarChartSquare from "~icons/solar/chart-square-bold-duotone";
import SolarPresentation from "~icons/solar/presentation-graph-bold-duotone";
import SolarGalleryDuo from "~icons/solar/gallery-bold-duotone";
import SolarVideoDuo from "~icons/solar/videocamera-bold-duotone";
import SolarMusicDuo from "~icons/solar/music-note-2-bold-duotone";
import SolarZip from "~icons/solar/zip-file-bold-duotone";
import SolarCode from "~icons/solar/code-bold-duotone";
import SolarFileDuo from "~icons/solar/file-bold-duotone";

type FileGlyph = { Comp: SvgComp; color: string };

const EXT_GLYPH: Record<string, FileGlyph> = {
  // Documents
  ".pdf": { Comp: SolarDocText, color: "#e8453c" },
  ".doc": { Comp: SolarDocText, color: "#2b7cd3" },
  ".docx": { Comp: SolarDocText, color: "#2b7cd3" },
  ".rtf": { Comp: SolarDocText, color: "#2b7cd3" },
  ".txt": { Comp: SolarDoc, color: "#64748b" },
  ".md": { Comp: SolarDoc, color: "#64748b" },
  // Spreadsheets
  ".xls": { Comp: SolarChartSquare, color: "#1f9d57" },
  ".xlsx": { Comp: SolarChartSquare, color: "#1f9d57" },
  ".csv": { Comp: SolarChartSquare, color: "#1f9d57" },
  // Presentations
  ".ppt": { Comp: SolarPresentation, color: "#e8730c" },
  ".pptx": { Comp: SolarPresentation, color: "#e8730c" },
  // Archives
  ".zip": { Comp: SolarZip, color: "#a855f7" },
  ".rar": { Comp: SolarZip, color: "#a855f7" },
  ".7z": { Comp: SolarZip, color: "#a855f7" },
  ".tar": { Comp: SolarZip, color: "#a855f7" },
  ".gz": { Comp: SolarZip, color: "#a855f7" },
  // Code
  ".js": { Comp: SolarCode, color: "#eab308" },
  ".ts": { Comp: SolarCode, color: "#3178c6" },
  ".tsx": { Comp: SolarCode, color: "#3178c6" },
  ".jsx": { Comp: SolarCode, color: "#eab308" },
  ".json": { Comp: SolarCode, color: "#64748b" },
  ".html": { Comp: SolarCode, color: "#e8453c" },
  ".css": { Comp: SolarCode, color: "#2b7cd3" },
  ".py": { Comp: SolarCode, color: "#3b82f6" },
  ".go": { Comp: SolarCode, color: "#06b6d4" },
  ".sh": { Comp: SolarCode, color: "#16a34a" },
  ".sql": { Comp: SolarCode, color: "#64748b" },
  ".xml": { Comp: SolarCode, color: "#e8730c" },
  ".yml": { Comp: SolarCode, color: "#64748b" },
  ".yaml": { Comp: SolarCode, color: "#64748b" },
};

const KIND_GLYPH: Record<string, FileGlyph> = {
  image: { Comp: SolarGalleryDuo, color: "#0ea5e9" },
  video: { Comp: SolarVideoDuo, color: "#ec4899" },
  audio: { Comp: SolarMusicDuo, color: "#8b5cf6" },
  archive: { Comp: SolarZip, color: "#a855f7" },
  document: { Comp: SolarDocText, color: "#2b7cd3" },
  other: { Comp: SolarFileDuo, color: "#64748b" },
};

// FileIcon renders a coloured, format-specific icon. `kind` is the coarse
// classification from the backend; `ext` (with leading dot) refines it.
export function FileIcon({ kind, ext, size = 32 }: { kind: string; ext?: string; size?: number }) {
  const e = (ext || "").toLowerCase();
  const glyph = (e && EXT_GLYPH[e]) || KIND_GLYPH[kind] || KIND_GLYPH.other;
  const Comp = glyph.Comp;
  return <Comp width={size} height={size} style={{ color: glyph.color }} />;
}

