# Changelog

## [1.8.0] - 2026-06-13

### Added

- **"Telegram Folder" — auto-import from native Telegram**: files uploaded directly into the configured storage channel from a native Telegram client (mobile/desktop) now appear automatically in a "Telegram Folder" in the drive. No toggle, no manual action — a background worker scans the channel every 5 minutes.
  - Imports are metadata-only (no download; streamed on-demand like any file), idempotent by `(channel_id, message_id)`, with a per-user cursor so incremental scans stay cheap.
  - Scans only the configured storage channel; read-only on Telegram; paced + FLOOD_WAIT-safe. Runs only when a channel is set and exactly one account exists (correct attribution to the Telegram session owner).
  - Doubles as recovery: if `metadata.db` is lost, re-scanning the channel brings the files back into the drive.

---

## [1.7.9] - 2026-06-12

### Added

- **File preview & viewing**: in-PWA viewer upgraded (image zoom/pan, video poster + seek, docx rendered via mammoth, "open in new tab"); share page now previews image/video/audio/pdf/docx inline behind the password gate.
- **Right-click + long-press context menu**: file/folder actions open via right-click (desktop) and long-press (mobile), not just the 3-dot button; menu repositions to stay in the viewport.
- **Dashboard stats**: home hero shows file count / folder count / total size / uptime (new `GET /v1/stats`), dropped the internal metadata.db path.
- **Community release prep**: rewritten README (one-click install per platform), full `docs/INSTALL.md`, docs set (overview/PDR, architecture, codebase, code standards), and one-click installers: `deploy/install.sh` (Linux/systemd), `deploy/install-docker.sh` (Docker), `deploy/install.ps1` (Windows).

### Fixed

- **Share download 403**: `RecordShareAccess` mismatched NULL `expires_at`/`max_downloads`, returning "limit reached/expired" for unlimited links — now COALESCE'd.
- **Share rate limit**: split a generous view limit (600/min) from the strict unlock limit (20/min) so viewing no longer trips "too many requests".
- **Icons**: unified Phosphor set via build-time inline SVG (offline-safe) with coloured per-format file icons; fixed Vietnamese text corruption from an earlier bulk edit.

---

## [1.7.8] - 2026-06-12

### Added

- **Standard thumbnail system for all file types**: images now scale with a high-quality bilinear kernel (was nearest-neighbor); videos get a real frame thumbnail via `ffmpeg`; PDFs render their first page via `pdftoppm` (poppler) or `mutool` (mupdf). External tools are optional — when missing, the file falls back to a clean kind icon, so uploads never fail over a preview. Thumbnails are (re)generated lazily on demand for files uploaded by older builds while the source is still cached locally.
- **Web Share Target**: once the PWA is installed, phones can share files into it from the OS share sheet (`POST /share-target`); shared files land in the drive root under the signed-in session.
- **Richer in-PWA viewer**: image viewer with zoom (wheel/buttons/double-tap) and pan, blurred-thumbnail placeholder while the full image loads; video uses the thumbnail as a poster and autoplays; "Open in new tab" alongside Download. Range-based streaming (seek) and Office/text/markdown/PDF viewing were already in place.

### UI

- Compact toolbar (segmented view toggle, icon-only secondary actions, single primary Upload), smaller/denser file & folder cards, removed redundant header text.
- Share modal no longer overflows on mobile (link wraps; actions stack).

---

## [1.7.7] - 2026-06-12

### Performance

- **Tens-of-thousands-file folder uploads**: dropping (or picking) a very large folder no longer freezes the tab or spikes memory. The previous path collected every file into one array before enqueuing — holding all `File` handles, blocking the UI during the scan, and only starting uploads after the whole tree was walked. Now the tree is streamed: an iterative (non-recursive) walk flushes files to the upload queue in batches of 200, yielding to the event loop between batches, so uploads begin while scanning continues and memory stays bounded. The folder picker and the queue's enqueue path were batched too (removed an O(n²) per-batch state rebuild).

---

## [1.7.6] - 2026-06-12

### Fixed

- **Remote mount showed stale local files (no app restart needed anymore)**: the mount backend was chosen once at agent startup, so opening the app (mounting local) and *then* connecting to a server left the running agent on the local backend — `T:` showed old local files instead of the paired server's files until restart. The drive now switches backend at runtime after pair/local/reset (`vfs.Manager.SwitchBackend` unmounts, swaps backend + mounter, remounts).
- **Drag & drop treated as browser download**: dropping a file onto the drive area let the browser open/download it instead of uploading. Now `preventDefault` runs on both `dragover` and `drop` at the window level for any file-carrying drag; the React handlers still enqueue the upload.
- **List flicker / "page reload" flash**: background revalidations (poll/focus/SSE/upload progress) toggled the loading state on every tick, blanking the list. Background refreshes now run silently; only navigation/first-load show the spinner. (DriveBrowser, StarredView)

### Notes

- Bundles all 1.7.5 fixes (large-file owner binding, search user scoping, orphan adoption on login, stale-resume 404, tus janitor, session-token hashing, folder-create race).

---

## [1.7.5] - 2026-06-12

### Fixed

- **Large files invisible after upload (data isolation)**: resumable (tus) uploads imported the assembled file on a background context and only read `user_id` from client metadata, which the PWA never sent — so files >32MB were saved with an empty owner and never appeared in the user-scoped listing. The tus handler now binds the authenticated user (from the request context, since `/v1/tus` is behind the auth gate) onto the upload at creation time. Also closes a cross-user data-isolation gap on multi-user instances.
- **Search leaked/showed orphaned files**: `Search()` did not filter by `user_id`, so empty-owner files appeared in search but not in the folder listing and could not be moved/renamed/trashed (those scope by user). Search now scopes by `user_id`, matching listing + operations.
- **Orphan repair on login**: when exactly one account exists, login now adopts any empty-owner data, repairing files imported by older agent builds. Scoped to single-user so it can never reassign another user's files.
- **Upload listing freshness**: the drive view now actively revalidates while uploads are in-flight (and once on settle), so imported files appear without waiting for SSE/poll.
- **tus stale-resume `HEAD 404`**: large uploads start clean instead of resuming a stored upload URL whose server-side temp file may already be imported/cleaned up.

### Security / Hardening

- **Session tokens hashed at rest**: stored as SHA-256 instead of plaintext, so a DB leak does not yield usable sessions. Existing sessions invalidate once (one re-login).
- **Duplicate folders on concurrent upload**: `getOrCreateFolder` lookup+insert is now serialized, so the 6-worker pool can't create duplicate folders with the same (parent, name, user) when uploading a folder.

### Added

- **tus temp janitor**: hourly sweep removes abandoned tus temp files older than 12h (interrupted uploads). Active uploads keep their mtime fresh and are never touched.

---

## [1.7.x] - 2026-06-11

### Desktop client + onboarding

- **Desktop onboarding (no CLI)**: native WebView2 setup window with "connect to existing server" / "run local server", server-URL test, pairing-code entry. State machine (`unset`/`local`/`remote`) persisted in `desktop.json`; auto-mount on start per saved mode.
- **Windows installer** `TelegramDriveSetup.exe`: bundles WinFsp, auto-detects PWA dir, `-H windowsgui` (no console window), valid `.ico` tray, Vietnamese (UTF-8 BOM), Innonet Agency branding. CI builds installer + tray + checksums; optional code-signing wired.
- **Tray "Open UI"** opens the configured server URL (remote) or localhost (local), resolved fresh per click.
- **Remote mount fix**: in remote mode the virtual drive mounts the paired server backend (was mounting the empty local DB → "not accessible").

### Upload reliability

- **Resumable chunked upload (tus protocol)** for files >32MB: 16MB chunks, single-stream, resumable across sessions, retry on error. Sidesteps reverse-proxy body-size limits. `RespectForwardedHeaders` so the upload URL is https behind a proxy.
- **Concurrent upload pool (6 workers)** with throttled UI (250ms flush), aggregate progress bar (speed/ETA), retry-failed, `beforeunload` guard, folder-scan feedback. Fixes lag and stalls on thousand-file folders.
- **Drop guard**: files dropped outside the dropzone no longer make the browser open the file.

### Realtime + UI

- **Stale-while-revalidate listing**: refresh on focus/visible/online + SSE + poll fallback. Fixes "uploaded to Telegram but not shown in PWA" caused by proxies buffering SSE. Agent sends anti-buffering headers + heartbeat on `/v1/events`.
- **Token-driven design system** redesign of the PWA, WCAG-AA contrast, responsive bottom-nav, fixed missing classes.
- **Vietnamese diacritics** fixed across installer, tray, and PWA.

### Debug + docs

- `GET /v1/debug/sync` + "Debug sync" view + `logs/sync.log` (JSON) for diagnosing upload/sync issues.
- New `docs/TROUBLESHOOTING.md`; `docs/DEPLOY.md` covers proxy upload limits, SSE buffering, cache sizing; `docs/CODE_SIGNING.md` for self-signed + checksums.

---

## [1.6.5] - 2026-05-21

### Features & Enhancements

- **REST API Enhancements (Actix-web & Rust)** — Fully implemented the comprehensive REST API extension in Rust/Actix-web with backwards-compatible response structures.
  - **Refined Folder Navigation**: Resolved `folder_id` query handling into three deterministic query states: all files when omitted, root-only when `?folder_id=`, and subfolder files when filtering specifically by a folder ID.
  - **Standardized Pagination Envelope**: Wrapped collections in a clean payload format featuring a `data` array, `pagination` metrics (`page`, `limit`, `total_items`, `total_pages`), and a `filters` echo block.
  - **Advanced Query Parameters**: Introduced server-side sorting (`sort_by`, `sort_order`) and robust filters for MIME type, file size bounds, and creation date ranges.
  - **Sparse Fieldsets**: Added a `?fields=` selector enabling clients to request specific metadata subsets to reduce bandwidth overhead.
  - **Bulk Operations & Global Search**: Added `POST /api/v1/files/bulk` for batch moves and deletes, and `GET /api/v1/files/search` supporting the full pagination envelope.
  - **Rate Limiting Integration**: Injected simulated API rate-limit headers (`X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`) to standard responses.

---

## [1.6.0] - 2026-05-21

### Features & Fixes

- **"Copy Telegram Link" Feature** — Added a right-click context menu option to copy raw `t.me` message links for files in public channels (`https://t.me/{username}/{message_id}`). If the channel is private, the item displays in a disabled state with a descriptive tooltip.
- **Tauri 2 Tokio Runtime Panic Fix** — Fixed the `there is no reactor running` panic caused by `tokio::task::spawn_blocking` executing outside of a Tokio runtime context within the Bandwidth Manager. Replaced the asynchronous task with a lightweight, synchronous write, resolving the panic completely.

---

## [1.5.0] - 2026-05-19

### Feature

- **VPN Optimizer & Proxy Configuration** — Added robust support for toggling VPN mode to optimize network connection timeouts, retry limits, backoff delays, adaptive polling, flood wait handling, and peer caches. Fully integrated proxy configuration (SOCKS5 and MTProto) to allow custom routing and bypass geo-blocks.

---

## [1.4.2] - 2026-05-18

### Feature

- **Folder Upload with Automatic Zipping** — Support uploading entire folders directly, automatically compressing them into highly-optimized zip archives before transfer.

---

## [1.1.7] - 2026-05-01

### Feature

- Added a donation button and popup modal to the main login screen to support the project via PayPal, Litecoin, and Bitcoin.

---

## [1.1.6] - 2026-04-28

### Fix

- Fixed process not terminating on Ctrl+C (SIGINT) when launched from a terminal.
  The Actix-web streaming server and grammers network runner were running on
  non-daemon threads with no shutdown signal wired to process exit, causing the
  application to hang indefinitely after the main window closed. The app now
  registers a RunEvent::Exit handler that gracefully stops both background
  services before the process exits.

---

## [1.1.5] - 2026-04-27

### Hotfix

- **CI fix: AppImage patch step now runs cleanly** — Replaced the fragile `grep -oP` Perl lookahead (which exited with code 2 under `set -euo pipefail`) with a safe `awk`-based `.desktop` file lookup. Added `APPIMAGE_EXTRACT_AND_RUN=1` so `appimagetool` doesn't require the FUSE kernel module on GitHub Actions runners.

---

## [1.1.4] - 2026-04-27

### Hotfix

- **Deeper AppImage EGL fix for Arch/rolling-release Linux** — Added a CI post-build patching step that strips the Ubuntu-bundled `libEGL`, `libGL`, `libGLdispatch`, `libGLX`, and `libGLESv2` from the AppImage squashfs and replaces the `AppRun` wrapper with one that: normalises the locale to `C.UTF-8`, sets `NO_AT_BRIDGE=1` to silence ATK warnings, auto-detects `EGL_PLATFORM` from `$WAYLAND_DISPLAY`/`$DISPLAY`, points GLVND at the system ICD vendor dirs, preloads the system `libEGL.so.1`, and orders `LD_LIBRARY_PATH` so host GPU drivers are always resolved before bundled stubs.

---

## [1.1.3] - 2026-04-27

### Hotfix

- **Fixed Arch Linux AppImage crash** — Resolved `EGL_BAD_ALLOC` error on Arch Linux (and other rolling-release distros) caused by bundled Mesa/EGL libraries conflicting with the host GPU driver stack. The app now automatically disables WebKitGTK's DMA-BUF renderer on Linux before the WebView initializes, with no impact to Windows or macOS builds.

---

## [1.0.4] - 2026-02-13

### Fixes

- Finally squashed the grid overlap bug for real. Cards were using CSS `aspect-[4/3]` to size themselves, but the virtualizer was computing row heights separately — at certain window widths these disagreed and rows would bleed into each other. Now both use the same explicit pixel height, so no more overlap regardless of how you resize the window.

### Cleanup

- Went through the whole codebase and ripped out every `console.log` / `console.error` we'd left in from debugging (16 of them). The one in `ErrorBoundary` stays since that's the whole point of an error boundary.
- Got rid of all `as any` casts on the frontend — everything's properly typed now.
- Ran Clippy and fixed all 7 warnings, including a couple of `collapsible_match` ones in `fs.rs` that needed manual refactoring.
- Dropped `clsx`, `tailwind-merge`, and `@tauri-apps/plugin-opener` from `package.json` — none of them were actually imported anywhere.
- General comment cleanup throughout.

---

## [1.0.3] - 2026-02-09

### Bug Fixes

- **Grid Spacing Fix** - Fixed cards overlapping in grid view
- **Dynamic Row Height** - Grid now properly calculates row height based on window size
- **Virtualizer Re-measurement** - Grid correctly updates when resizing window

---

## [1.0.2] - 2026-02-07

### Automated Release Pipeline

- **GitHub Actions Workflow** - Automatic builds triggered on version tags
- **Cross-Platform Builds** - Windows, Linux, macOS (Intel + ARM) built in parallel
- **Signed Updates** - All builds signed with Ed25519 for secure auto-updates
- **Automatic Publishing** - Releases published to GitHub automatically

---

## [1.0.1] - 2026-02-07

### Auto-Update System

- **Automatic Update Checks** - App checks for updates 5 seconds after startup
- **Update Banner** - Beautiful animated banner when new version available
- **One-Click Updates** - Download and install updates with progress indicator
- **Cross-Platform** - Windows, Mac, and Linux users get platform-specific updates

### 🔧 Technical

- Added Tauri updater plugin with Ed25519 signing
- Created `useUpdateCheck` hook for update lifecycle management
- Added `UpdateBanner` component with download progress

---

## [1.0.0] - 2026-02-06 🎉

### First Stable Release

Telegram Drive is now production-ready! This release focuses on performance, reliability, and user experience polish.

### ✨ New Features

- **Virtual Scrolling** - Smooth performance with folders containing 1000+ files
- **Inline Thumbnails** - Image files now display thumbnails directly in the file grid
- **Thumbnail Caching** - Thumbnails are cached locally for instant loading on revisit
- **API Setup Help Guide** - Step-by-step modal explaining how to get Telegram API credentials

### 🚀 Performance Improvements

- Grid and list views now only render visible items (virtualized)
- Responsive column layout adapts to window width
- Lazy loading of thumbnails to reduce initial load time

### 🎨 UI/UX Improvements

- Refined grid spacing (6px gaps between cards)
- Gradient overlay on thumbnail cards for text readability
- Improved light mode support across all components

### 🔧 Technical

- Added `@tanstack/react-virtual` for virtualization
- Separate thumbnail cache directory (`app_data_dir/thumbnails/`)
- FileTypeIcon now supports multiple sizes

---

## [0.6.0] - 2026-02-05

### Reliability Update

- Session persistence (window state, UI state, active folder)
- Network resilience with connection status indicator
- Queue persistence for uploads/downloads
- Light mode UI fixes

---

## [0.5.0] - 2026-02-04

### Drag & Drop Update

- Stable hybrid drag-drop system
- External drop blocker
- GitHub Actions workflow fixes

---

## [0.4.0] - 2026-02-01

### Media & Performance

- Audio/Video streaming player
- Global search filter
- Internal drag & drop between folders
