package secret

import (
	"bytes"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	key, err := genKey(t)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("hello telegram drive")
	enc, err := Encrypt(key, msg)
	if err != nil {
		t.Fatal(err)
	}
	if !HasMagic(enc) {
		t.Fatalf("encrypted blob missing magic: %x", enc[:4])
	}
	dec, err := Decrypt(key, enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dec, msg) {
		t.Fatalf("roundtrip mismatch: got %q want %q", dec, msg)
	}
}

func TestDecryptDetectsLegacyPlain(t *testing.T) {
	key, err := genKey(t)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt(key, []byte(`{"plain":"json"}`)); err != ErrLegacyPlain {
		t.Fatalf("expected ErrLegacyPlain, got %v", err)
	}
}

func TestDecryptWrongKeyFails(t *testing.T) {
	keyA, _ := genKey(t)
	keyB, _ := genKey(t)
	enc, err := Encrypt(keyA, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt(keyB, enc); err == nil {
		t.Fatal("expected auth failure with wrong key")
	}
}

func TestLoadKeyFromHexEnv(t *testing.T) {
	t.Setenv(EnvVar, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	key, err := LoadKey()
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != KeySize {
		t.Fatalf("expected 32 bytes, got %d", len(key))
	}
}

func TestLoadKeyEmpty(t *testing.T) {
	t.Setenv(EnvVar, "")
	key, err := LoadKey()
	if err != nil {
		t.Fatal(err)
	}
	if key != nil {
		t.Fatalf("expected nil key when env empty")
	}
}

func TestLoadKeyInvalid(t *testing.T) {
	t.Setenv(EnvVar, "not-a-real-key")
	if _, err := LoadKey(); err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func genKey(t *testing.T) ([]byte, error) {
	t.Helper()
	hexKey, err := Generate()
	if err != nil {
		return nil, err
	}
	t.Setenv(EnvVar, hexKey)
	return LoadKey()
}
