package vault

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"
)

// Secret represents a stored secret with its value and metadata.
type Secret struct {
	// Value is the primary secret value as a string.
	Value string `json:"value,omitempty"`

	// ValueBytes is the secret value as bytes, for binary secrets.
	// If both Value and ValueBytes are set, ValueBytes takes precedence.
	ValueBytes []byte `json:"valueBytes,omitempty"`

	// Fields contains additional named fields for multi-field secrets.
	// Common for password managers that store username, password, URL, etc.
	Fields map[string]string `json:"fields,omitempty"`

	// Metadata contains additional information about the secret.
	Metadata Metadata `json:"metadata,omitempty"`
}

// GetField returns a field value, falling back to the main Value if the field
// doesn't exist and the field name is empty or "value".
func (s *Secret) GetField(name string) string {
	if name == "" || name == "value" {
		return s.Value
	}
	if s.Fields != nil {
		if v, ok := s.Fields[name]; ok {
			return v
		}
	}
	return ""
}

// SetField sets a field value. If the field name is empty or "value",
// it sets the main Value field.
func (s *Secret) SetField(name, value string) {
	if name == "" || name == "value" {
		s.Value = value
		return
	}
	if s.Fields == nil {
		s.Fields = make(map[string]string)
	}
	s.Fields[name] = value
}

// Redacted is what a [Secret] prints as. Formatting a secret with fmt or
// logging it with log/slog never shows its value; call [Secret.Reveal] or
// [Secret.Bytes] to read it.
//
// Secret deliberately has no String method. It used to return the value, so
// code that called it to read the value now fails to compile instead of
// silently receiving the redaction marker.
const Redacted = "[REDACTED]"

// Reveal returns the primary value of the secret: ValueBytes as a string if
// set, otherwise Value. Use it where the value is needed, never to log it.
func (s *Secret) Reveal() string {
	if len(s.ValueBytes) > 0 {
		return string(s.ValueBytes)
	}
	return s.Value
}

// GoString implements [fmt.GoStringer], so %#v is redacted too.
func (s Secret) GoString() string { return Redacted }

// Format implements [fmt.Formatter]. Every verb, including %v, %+v, %#v, %s,
// %q and %x, prints [Redacted], so neither the value nor the fields of a
// Secret or *Secret can be printed through fmt.
func (s Secret) Format(f fmt.State, _ rune) {
	// Format cannot return an error, and the fmt.State that fmt passes in
	// writes to an in-memory buffer that does not fail.
	if _, err := io.WriteString(f, Redacted); err != nil {
		return
	}
}

// LogValue implements [slog.LogValuer], so a secret logged with log/slog
// records [Redacted].
func (s Secret) LogValue() slog.Value { return slog.StringValue(Redacted) }

// Bytes returns the secret value as bytes.
func (s *Secret) Bytes() []byte {
	if len(s.ValueBytes) > 0 {
		return s.ValueBytes
	}
	return []byte(s.Value)
}

// Metadata contains additional information about a secret.
type Metadata struct {
	// CreatedAt is when the secret was created.
	CreatedAt *Timestamp `json:"createdAt,omitempty"`

	// ModifiedAt is when the secret was last modified.
	ModifiedAt *Timestamp `json:"modifiedAt,omitempty"`

	// ExpiresAt is when the secret expires, if applicable.
	ExpiresAt *Timestamp `json:"expiresAt,omitempty"`

	// Version is the version identifier of the secret.
	Version string `json:"version,omitempty"`

	// Tags are key-value pairs for categorization.
	Tags map[string]string `json:"tags,omitempty"`

	// Labels are simple string labels.
	Labels []string `json:"labels,omitempty"`

	// Provider is the name of the provider that stored this secret.
	Provider string `json:"provider,omitempty"`

	// Path is the path where this secret is stored.
	Path string `json:"path,omitempty"`

	// Extra contains provider-specific metadata.
	Extra map[string]any `json:"extra,omitempty"`
}

// Timestamp wraps time.Time to provide custom JSON marshaling.
type Timestamp struct {
	time.Time
}

// NewTimestamp creates a new Timestamp from a time.Time.
func NewTimestamp(t time.Time) *Timestamp {
	return &Timestamp{Time: t}
}

// Now returns a Timestamp for the current time.
func Now() *Timestamp {
	return &Timestamp{Time: time.Now()}
}

// MarshalJSON implements json.Marshaler.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Time.Format(time.RFC3339))
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	t.Time = parsed
	return nil
}

// SecretRef is a URI-style reference to a secret.
// Examples:
//
//	op://vault/item/field          (1Password)
//	keychain://service/account     (macOS Keychain)
//	env://VAR_NAME                 (Environment variable)
//	file:///path/to/secret         (File)
//	vault://secret/path#field      (HashiCorp Vault)
//	aws-sm://secret-name#key       (AWS Secrets Manager)
//	gcp-sm://project/secret        (GCP Secret Manager)
type SecretRef string

// Scheme returns the scheme portion of the secret reference (e.g., "op", "env").
func (r SecretRef) Scheme() string {
	s := string(r)
	for i, c := range s {
		if c == ':' {
			return s[:i]
		}
	}
	return ""
}

// Path returns the path portion of the secret reference (after ://).
func (r SecretRef) Path() string {
	s := string(r)
	for i := 0; i < len(s)-2; i++ {
		if s[i] == ':' && s[i+1] == '/' && s[i+2] == '/' {
			path := s[i+3:]
			// Remove fragment
			for j, c := range path {
				if c == '#' {
					return path[:j]
				}
			}
			return path
		}
	}
	return s
}

// Fragment returns the fragment portion of the secret reference (after #).
func (r SecretRef) Fragment() string {
	s := string(r)
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '#' {
			return s[i+1:]
		}
	}
	return ""
}

// String returns the string representation of the secret reference.
func (r SecretRef) String() string {
	return string(r)
}
