package drive

import (
	"errors"
	"testing"
	"time"
)

func TestBackoffForFloodWait(t *testing.T) {
	got := backoffFor(1, errors.New("rpc error code 420: FLOOD_WAIT_42"))
	want := 42 * time.Second
	if got != want {
		t.Fatalf("FLOOD_WAIT parse: got %v want %v", got, want)
	}
}

func TestBackoffForGenericIncreases(t *testing.T) {
	first := backoffFor(1, errors.New("transient"))
	third := backoffFor(3, errors.New("transient"))
	if first >= third {
		t.Fatalf("expected backoff to grow with attempts, got first=%v third=%v", first, third)
	}
}

func TestBackoffForCaps(t *testing.T) {
	last := backoffFor(99, errors.New("transient"))
	want := backoffSteps[len(backoffSteps)-1]
	if last != want {
		t.Fatalf("expected cap to %v, got %v", want, last)
	}
}

func TestUploadGateRetryWindow(t *testing.T) {
	svc := &Service{}
	svc.scheduleRetry("file-1", errors.New("rpc error code 420: FLOOD_WAIT_5"))
	if svc.uploadGateAllow(nil, "file-1") {
		t.Fatalf("expected gate to block while in flood wait")
	}
	svc.clearRetry("file-1")
	if !svc.uploadGateAllow(nil, "file-1") {
		t.Fatalf("expected gate to allow after clear")
	}
}
