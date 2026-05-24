package drive

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func (s *Service) HLSStream(ctx context.Context, fileID string, w http.ResponseWriter) error {
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg chưa được cài trên máy")
	}
	stream, err := s.GetStreamableFile(ctx, fileID)
	if err != nil {
		return err
	}
	if stream.Source != StreamFromCache || stream.LocalPath == "" {
		return fmt.Errorf("HLS stream cần file đã có cache local")
	}
	args := []string{
		"-loglevel", "error",
		"-i", stream.LocalPath,
		"-c:v", "copy",
		"-c:a", "copy",
		"-f", "mpegts",
		"pipe:1",
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = w
	cmd.Stderr = devnull{}
	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "no-store")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg stream: %w", err)
	}
	return nil
}

type devnull struct{}

func (devnull) Write(p []byte) (int, error) { return len(p), nil }

func ensureFFmpegArgs(args []string, extraOffsetSeconds int) []string {
	if extraOffsetSeconds <= 0 {
		return args
	}
	prefix := []string{"-ss", strconv.Itoa(extraOffsetSeconds)}
	return append(prefix, args...)
}

func DefaultOutputName(localPath string) string {
	base := filepath.Base(localPath)
	if dot := strings.LastIndex(base, "."); dot > 0 {
		base = base[:dot]
	}
	return base + ".ts"
}
