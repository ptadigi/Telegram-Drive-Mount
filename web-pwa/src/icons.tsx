// Central icon module. The whole PWA imports icons from here (via the
// "lucide-react" Vite alias) so we have ONE consistent icon set — Phosphor
// (duotone) — and a single place to swap sets later.
//
// Icons compile to inline SVG at build time via unplugin-icons (no runtime
// network fetch — safe for an offline/self-hosted PWA). Each glyph is wrapped
// so call sites keep the lucide-style API (`size`, `className`, `color`;
// `strokeWidth` is accepted but ignored).
import type { ComponentType, SVGProps } from "react";

import PhArchive from "~icons/ph/archive-duotone";
import PhCalendar from "~icons/ph/calendar-dots-duotone";
import PhCheckCircle from "~icons/ph/check-circle-duotone";
import PhCaretDown from "~icons/ph/caret-down-duotone";
import PhCaretRight from "~icons/ph/caret-right-duotone";
import PhCaretUp from "~icons/ph/caret-up-duotone";
import PhClock from "~icons/ph/clock-duotone";
import PhCloud from "~icons/ph/cloud-duotone";
import PhCopy from "~icons/ph/copy-duotone";
import PhDatabase from "~icons/ph/database-duotone";
import PhDownload from "~icons/ph/download-simple-duotone";
import PhExternal from "~icons/ph/arrow-square-out-duotone";
import PhFileAudio from "~icons/ph/file-audio-duotone";
import PhFileText from "~icons/ph/file-text-duotone";
import PhFileVideo from "~icons/ph/file-video-duotone";
import PhFolder from "~icons/ph/folder-duotone";
import PhFolderNotchOpen from "~icons/ph/folder-notch-open-duotone";
import PhFolderOpen from "~icons/ph/folder-open-duotone";
import PhFolderPlus from "~icons/ph/folder-plus-duotone";
import PhFolderDashed from "~icons/ph/folder-simple-dashed-duotone";
import PhGlobe from "~icons/ph/globe-duotone";
import PhHardDrive from "~icons/ph/hard-drive-duotone";
import PhHouse from "~icons/ph/house-duotone";
import PhImage from "~icons/ph/image-duotone";
import PhInfo from "~icons/ph/info-duotone";
import PhKey from "~icons/ph/key-duotone";
import PhLaptop from "~icons/ph/laptop-duotone";
import PhSquaresFour from "~icons/ph/squares-four-duotone";
import PhLink from "~icons/ph/link-duotone";
import PhList from "~icons/ph/list-duotone";
import PhSpinner from "~icons/ph/spinner-gap-duotone";
import PhLock from "~icons/ph/lock-duotone";
import PhDotsThreeV from "~icons/ph/dots-three-vertical-duotone";
import PhPause from "~icons/ph/pause-duotone";
import PhPencil from "~icons/ph/pencil-simple-duotone";
import PhPhone from "~icons/ph/phone-duotone";
import PhPlay from "~icons/ph/play-duotone";
import PhPlusCircle from "~icons/ph/plus-circle-duotone";
import PhQr from "~icons/ph/qr-code-duotone";
import PhArrowsClockwise from "~icons/ph/arrows-clockwise-duotone";
import PhArrowCCW from "~icons/ph/arrow-counter-clockwise-duotone";
import PhArrowCW from "~icons/ph/arrow-clockwise-duotone";
import PhMagnifier from "~icons/ph/magnifying-glass-duotone";
import PhHardDrives from "~icons/ph/hard-drives-duotone";
import PhGear from "~icons/ph/gear-duotone";
import PhShareNetwork from "~icons/ph/share-network-duotone";
import PhShieldCheck from "~icons/ph/shield-check-duotone";
import PhShieldSlash from "~icons/ph/shield-slash-duotone";
import PhSparkle from "~icons/ph/sparkle-duotone";
import PhStar from "~icons/ph/star-duotone";
import PhTrash from "~icons/ph/trash-duotone";
import PhUpload from "~icons/ph/upload-simple-duotone";
import PhCloudArrowUp from "~icons/ph/cloud-arrow-up-duotone";
import PhWifi from "~icons/ph/wifi-high-duotone";
import PhX from "~icons/ph/x-duotone";
import PhXCircle from "~icons/ph/x-circle-duotone";
import PhZoomIn from "~icons/ph/magnifying-glass-plus-duotone";
import PhZoomOut from "~icons/ph/magnifying-glass-minus-duotone";

export type IconProps = SVGProps<SVGSVGElement> & {
  size?: number | string;
  strokeWidth?: number | string;
};

type SvgComp = ComponentType<SVGProps<SVGSVGElement>>;

function wrap(Comp: SvgComp) {
  return function Icon({ size = 24, strokeWidth: _sw, width, height, ...rest }: IconProps) {
    return <Comp width={width ?? size} height={height ?? size} {...rest} />;
  };
}

export const Archive = wrap(PhArchive);
export const CalendarClock = wrap(PhCalendar);
export const CheckCircle2 = wrap(PhCheckCircle);
export const ChevronDown = wrap(PhCaretDown);
export const ChevronRight = wrap(PhCaretRight);
export const ChevronUp = wrap(PhCaretUp);
export const Clock3 = wrap(PhClock);
export const Cloud = wrap(PhCloud);
export const Copy = wrap(PhCopy);
export const Database = wrap(PhDatabase);
export const Download = wrap(PhDownload);
export const ExternalLink = wrap(PhExternal);
export const FileAudio = wrap(PhFileAudio);
export const FileText = wrap(PhFileText);
export const FileVideo = wrap(PhFileVideo);
export const Folder = wrap(PhFolder);
export const FolderInput = wrap(PhFolderNotchOpen);
export const FolderOpen = wrap(PhFolderOpen);
export const FolderPlus = wrap(PhFolderPlus);
export const FolderSync = wrap(PhFolderDashed);
export const FolderUp = wrap(PhFolderNotchOpen);
export const Globe = wrap(PhGlobe);
export const HardDrive = wrap(PhHardDrive);
export const Home = wrap(PhHouse);
export const Image = wrap(PhImage);
export const Info = wrap(PhInfo);
export const KeyRound = wrap(PhKey);
export const Laptop2 = wrap(PhLaptop);
export const LayoutGrid = wrap(PhSquaresFour);
export const Link2 = wrap(PhLink);
export const List = wrap(PhList);
export const Loader2 = wrap(PhSpinner);
export const Lock = wrap(PhLock);
export const MoreVertical = wrap(PhDotsThreeV);
export const Pause = wrap(PhPause);
export const Pencil = wrap(PhPencil);
export const Phone = wrap(PhPhone);
export const Play = wrap(PhPlay);
export const Plus = wrap(PhPlusCircle);
export const QrCode = wrap(PhQr);
export const RefreshCw = wrap(PhArrowsClockwise);
export const RotateCcw = wrap(PhArrowCCW);
export const RotateCw = wrap(PhArrowCW);
export const Search = wrap(PhMagnifier);
export const Server = wrap(PhHardDrives);
export const Settings = wrap(PhGear);
export const Share2 = wrap(PhShareNetwork);
export const ShieldCheck = wrap(PhShieldCheck);
export const ShieldOff = wrap(PhShieldSlash);
export const Sparkles = wrap(PhSparkle);
export const Star = wrap(PhStar);
export const Trash2 = wrap(PhTrash);
export const Upload = wrap(PhUpload);
export const UploadCloud = wrap(PhCloudArrowUp);
export const Wifi = wrap(PhWifi);
export const X = wrap(PhX);
export const XCircle = wrap(PhXCircle);
export const ZoomIn = wrap(PhZoomIn);
export const ZoomOut = wrap(PhZoomOut);

// ===================== Coloured file-type icons =====================
// Google-Drive-style: each format gets a distinct duotone glyph + colour.
import PhFileDoc from "~icons/ph/file-doc-duotone";
import PhFileXls from "~icons/ph/file-xls-duotone";
import PhFilePpt from "~icons/ph/file-ppt-duotone";
import PhFileZip from "~icons/ph/file-zip-duotone";
import PhFileCode from "~icons/ph/file-code-duotone";
import PhFilePdf from "~icons/ph/file-pdf-duotone";
import PhFile from "~icons/ph/file-duotone";

type FileGlyph = { Comp: SvgComp; color: string };

const EXT_GLYPH: Record<string, FileGlyph> = {
  ".pdf": { Comp: PhFilePdf, color: "#e8453c" },
  ".doc": { Comp: PhFileDoc, color: "#2b7cd3" },
  ".docx": { Comp: PhFileDoc, color: "#2b7cd3" },
  ".rtf": { Comp: PhFileDoc, color: "#2b7cd3" },
  ".txt": { Comp: PhFileText, color: "#64748b" },
  ".md": { Comp: PhFileText, color: "#64748b" },
  ".xls": { Comp: PhFileXls, color: "#1f9d57" },
  ".xlsx": { Comp: PhFileXls, color: "#1f9d57" },
  ".csv": { Comp: PhFileXls, color: "#1f9d57" },
  ".ppt": { Comp: PhFilePpt, color: "#e8730c" },
  ".pptx": { Comp: PhFilePpt, color: "#e8730c" },
  ".zip": { Comp: PhFileZip, color: "#a855f7" },
  ".rar": { Comp: PhFileZip, color: "#a855f7" },
  ".7z": { Comp: PhFileZip, color: "#a855f7" },
  ".tar": { Comp: PhFileZip, color: "#a855f7" },
  ".gz": { Comp: PhFileZip, color: "#a855f7" },
  ".js": { Comp: PhFileCode, color: "#eab308" },
  ".ts": { Comp: PhFileCode, color: "#3178c6" },
  ".tsx": { Comp: PhFileCode, color: "#3178c6" },
  ".jsx": { Comp: PhFileCode, color: "#eab308" },
  ".json": { Comp: PhFileCode, color: "#64748b" },
  ".html": { Comp: PhFileCode, color: "#e8453c" },
  ".css": { Comp: PhFileCode, color: "#2b7cd3" },
  ".py": { Comp: PhFileCode, color: "#3b82f6" },
  ".go": { Comp: PhFileCode, color: "#06b6d4" },
  ".sh": { Comp: PhFileCode, color: "#16a34a" },
  ".sql": { Comp: PhFileCode, color: "#64748b" },
  ".xml": { Comp: PhFileCode, color: "#e8730c" },
  ".yml": { Comp: PhFileCode, color: "#64748b" },
  ".yaml": { Comp: PhFileCode, color: "#64748b" },
};

const KIND_GLYPH: Record<string, FileGlyph> = {
  image: { Comp: PhImage, color: "#0ea5e9" },
  video: { Comp: PhFileVideo, color: "#ec4899" },
  audio: { Comp: PhFileAudio, color: "#8b5cf6" },
  archive: { Comp: PhFileZip, color: "#a855f7" },
  document: { Comp: PhFileDoc, color: "#2b7cd3" },
  other: { Comp: PhFile, color: "#64748b" },
};

export function FileIcon({ kind, ext, size = 32 }: { kind: string; ext?: string; size?: number }) {
  const e = (ext || "").toLowerCase();
  const glyph = (e && EXT_GLYPH[e]) || KIND_GLYPH[kind] || KIND_GLYPH.other;
  const Comp = glyph.Comp;
  return <Comp width={size} height={size} style={{ color: glyph.color }} />;
}
