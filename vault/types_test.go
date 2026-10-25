package vault

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

const plaintext = "hunter2-plaintext"

func newTestSecret() *Secret {
	return &Secret{
		Value:  plaintext,
		Fields: map[string]string{"password": plaintext},
	}
}

func TestSecretFormattingIsRedacted(t *testing.T) {
	s := newTestSecret()

	outputs := map[string]string{
		"GoString":       s.GoString(),
		"Sprint(ptr)":    fmt.Sprint(s),
		"Sprint(value)":  fmt.Sprint(*s),
		"%v":             fmt.Sprintf("%v", s),
		"%+v":            fmt.Sprintf("%+v", s),
		"%#v":            fmt.Sprintf("%#v", s),
		"%s":             fmt.Sprintf("%s", s),
		"%q":             fmt.Sprintf("%q", s),
		"%x":             fmt.Sprintf("%x", s),
		"%v value":       fmt.Sprintf("%v", *s),
		"%+v value":      fmt.Sprintf("%+v", *s),
		"%v in slice":    fmt.Sprintf("%v", []*Secret{s}),
		"%+v in struct":  fmt.Sprintf("%+v", struct{ S *Secret }{s}),
		"error wrapping": fmt.Errorf("loading %v: %w", s, errors.New("boom")).Error(),
	}
	for name, out := range outputs {
		if strings.Contains(out, plaintext) || strings.Contains(out, fmt.Sprintf("%x", plaintext)) {
			t.Errorf("%s leaked the value: %q", name, out)
		}
		if !strings.Contains(out, Redacted) {
			t.Errorf("%s = %q, want it to contain %q", name, out, Redacted)
		}
	}
}

func TestSecretSlogIsRedacted(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	s := newTestSecret()
	logger.Info("loaded", "secret", s, "value", *s)
	logger.Info("loaded", slog.Any("secret", s))

	out := buf.String()
	if strings.Contains(out, plaintext) {
		t.Fatalf("slog output leaked the value: %s", out)
	}
	if !strings.Contains(out, Redacted) {
		t.Fatalf("slog output = %s, want %q", out, Redacted)
	}
}

func TestSecretReveal(t *testing.T) {
	if got := (&Secret{Value: "text"}).Reveal(); got != "text" {
		t.Fatalf("Reveal = %q, want %q", got, "text")
	}
	if got := (&Secret{Value: "text", ValueBytes: []byte("bytes")}).Reveal(); got != "bytes" {
		t.Fatalf("Reveal with ValueBytes = %q, want %q", got, "bytes")
	}
}

// JSON is how providers persist and transfer secrets, so it must keep the value.
func TestSecretJSONKeepsValue(t *testing.T) {
	b, err := json.Marshal(newTestSecret())
	if err != nil {
		t.Fatal(err)
	}
	var got Secret
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Value != plaintext || got.Fields["password"] != plaintext {
		t.Fatalf("JSON round trip lost the value: %s", b)
	}
}
