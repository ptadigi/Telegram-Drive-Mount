package drive

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/gif"  // register GIF decoder
	_ "image/png"  // register PNG decoder
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"golang.org/x/image/draw"
)

// thumbMaxSize is the longest-edge size (px) of generated thumbnails. 320 keeps
// grid cards crisp on hi-dpi phones without bloating the cache.
const thumbMaxSize = 320

// thumbnailKinds reports whether a file kind can have a real (bitmap) thumbnail.
// "document" only qualifies for PDFs; that is handled by extension in the
// generator. Everything else falls back to a kind icon in the UI.
func canThumbnail(kind, ext string) bool {
	switch kind {
	case "image", "video":
		return true
	case "document":
		return ext == ".pdf"
	default:
		return false
	}
}

// generateThumbnail produces a JPEG thumbnail for the file at localPath and
// returns its path. It is best-effort: callers treat an error as "no thumbnail"
// and the UI shows a kind icon. Video/PDF rendering needs external binaries
// (ffmpeg / pdftoppm or mutool); when they are missing we return an error so
// the caller marks the preview unsupported rather than failing the upload.
func (s *Service) generateThumbnail(fileID, localPath, kind, ext string) (string, error) {
	switch {
	case kind == "image":
		return s.thumbnailFromImage(fileID, localPath)
	case kind == "video":
		return s.thumbnailFromVideo(fileID, localPath)
	case kind == "document" && ext == ".pdf":
		return s.thumbnailFromPDF(fileID, localPath)
	default:
		return "", fmt.Errorf("kind %q không hỗ trợ thumbnail", kind)
	}
}

func (s *Service) thumbDir() (string, error) {
	dir := filepath.Join(s.dataDir, "thumbs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// thumbnailFromImage decodes any registered image format and writes a
// high-quality (bilinear-scaled) JPEG thumbnail.
func (s *Service) thumbnailFromImage(fileID, localPath string) (string, error) {
	source, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer source.Close()
	img, _, err := image.Decode(source)
	if err != nil {
		return "", err
	}
	return s.encodeThumb(fileID, scaleImage(img, thumbMaxSize))
}

// thumbnailFromVideo extracts a representative frame with ffmpeg, then scales it
// down. Requires ffmpeg on PATH; otherwise returns an error (caller falls back
// to an icon). Best-effort, time-boxed so a broken file can't hang uploads.
func (s *Service) thumbnailFromVideo(fileID, localPath string) (string, error) {
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("ffmpeg chưa cài, bỏ qua thumbnail video")
	}
	dir, err := s.thumbDir()
	if err != nil {
		return "", err
	}
	tmp := filepath.Join(dir, fileID+".src.jpg")
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	// Seek to 1s (avoids black intro frames), grab one frame, downscale in ffmpeg.
	cmd := exec.CommandContext(ctx, bin,
		"-y", "-ss", "1", "-i", localPath,
		"-frames:v", "1",
		"-vf", fmt.Sprintf("scale='min(%d,iw)':-2", thumbMaxSize),
		"-q:v", "3",
		tmp,
	)
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("ffmpeg trích frame thất bại: %w", err)
	}
	defer os.Remove(tmp)
	// Re-encode through our pipeline for a consistent name + size cap.
	src, err := os.Open(tmp)
	if err != nil {
		return "", err
	}
	defer src.Close()
	img, _, err := image.Decode(src)
	if err != nil {
		return "", err
	}
	return s.encodeThumb(fileID, scaleImage(img, thumbMaxSize))
}

// thumbnailFromPDF renders the first page using pdftoppm (poppler) or mutool
// (mupdf) when available. Returns an error if neither is installed.
func (s *Service) thumbnailFromPDF(fileID, localPath string) (string, error) {
	dir, err := s.thumbDir()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	if bin, lookErr := exec.LookPath("pdftoppm"); lookErr == nil {
		prefix := filepath.Join(dir, fileID+".pdfsrc")
		// -f 1 -l 1: first page only; -r 96 dpi; -jpeg output -> <prefix>-1.jpg
		cmd := exec.CommandContext(ctx, bin, "-jpeg", "-r", "96", "-f", "1", "-l", "1", "-singlefile", localPath, prefix)
		if err := cmd.Run(); err == nil {
			out := prefix + ".jpg"
			if _, statErr := os.Stat(out); statErr == nil {
				defer os.Remove(out)
				return s.thumbnailFromImage(fileID, out)
			}
		}
	}
	if bin, lookErr := exec.LookPath("mutool"); lookErr == nil {
		out := filepath.Join(dir, fileID+".pdfsrc.png")
		cmd := exec.CommandContext(ctx, bin, "draw", "-o", out, "-r", "96", localPath, "1")
		if err := cmd.Run(); err == nil {
			if _, statErr := os.Stat(out); statErr == nil {
				defer os.Remove(out)
				return s.thumbnailFromImage(fileID, out)
			}
		}
	}
	return "", fmt.Errorf("không có pdftoppm/mutool để render PDF")
}

// encodeThumb writes a JPEG thumbnail for fileID and returns its path.
func (s *Service) encodeThumb(fileID string, img image.Image) (string, error) {
	dir, err := s.thumbDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, fileID+".jpg")
	target, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer target.Close()
	if err := jpeg.Encode(target, img, &jpeg.Options{Quality: 82}); err != nil {
		return "", err
	}
	return path, nil
}

// scaleImage downscales src so its longest edge is at most maxSize, using a
// high-quality bilinear kernel. Images already within bounds are returned
// unchanged.
func scaleImage(src image.Image, maxSize int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return src
	}
	if w <= maxSize && h <= maxSize {
		return src
	}
	scale := float64(maxSize) / float64(w)
	if h > w {
		scale = float64(maxSize) / float64(h)
	}
	dw := int(float64(w) * scale)
	dh := int(float64(h) * scale)
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}
