package tunnel

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
	"time"
)

type Status struct {
	Active    bool   `json:"active"`
	URL       string `json:"url,omitempty"`
	StartedAt int64  `json:"started_at,omitempty"`
	LastError string `json:"last_error,omitempty"`
	Binary    string `json:"binary,omitempty"`
	LocalPort int    `json:"local_port,omitempty"`
	IsBuiltIn bool   `json:"is_builtin"`
}

type Listener interface {
	OnTunnelStatus(status Status)
}

type Service struct {
	mu       sync.Mutex
	cmd      *exec.Cmd
	cancel   context.CancelFunc
	status   Status
	listener Listener
}

func New(listener Listener) *Service {
	return &Service{listener: listener}
}

func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *Service) Start(localPort int) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status.Active {
		return s.status, nil
	}
	binary, err := exec.LookPath("cloudflared")
	if err != nil {
		s.status = Status{LastError: "Không tìm thấy cloudflared trong PATH. Hãy cài cloudflared và thử lại."}
		s.notify()
		return s.status, errors.New(s.status.LastError)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, binary, "tunnel", "--no-autoupdate", "--url", fmt.Sprintf("http://127.0.0.1:%d", localPort))
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return Status{}, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return Status{}, err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		s.status = Status{LastError: err.Error()}
		s.notify()
		return s.status, err
	}
	s.cmd = cmd
	s.cancel = cancel
	s.status = Status{Active: true, StartedAt: time.Now().Unix(), Binary: binary, LocalPort: localPort}
	s.notify()
	go s.parseOutput(stdout)
	go s.parseOutput(stderr)
	go s.waitProcess(cmd)
	return s.status, nil
}

func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.status.Active {
		return
	}
	if s.cancel != nil {
		s.cancel()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	s.status = Status{LastError: "Đã dừng Cloudflare Tunnel"}
	s.notify()
}

func (s *Service) parseOutput(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)
	urlPattern := regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)
	for scanner.Scan() {
		line := scanner.Text()
		if match := urlPattern.FindString(line); match != "" {
			s.mu.Lock()
			s.status.URL = match
			s.status.LastError = ""
			s.notify()
			s.mu.Unlock()
		}
	}
}

func (s *Service) waitProcess(cmd *exec.Cmd) {
	err := cmd.Wait()
	s.mu.Lock()
	s.status.Active = false
	if err != nil {
		s.status.LastError = err.Error()
	}
	s.notify()
	s.mu.Unlock()
}

func (s *Service) notify() {
	if s.listener == nil {
		return
	}
	s.listener.OnTunnelStatus(s.status)
}

func ParsePort(value string) int {
	port, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return port
}
