package recorder

import (
	"errors"
	"os"
	"path"
	"testing"
)

// TestParseMetaArgs verifies split-on-first-'=' and RFC 3986 PathUnescape
// decoding of both key and value, including %3D (literal '='), %20 (space), a
// non-ASCII byte, and the '+'-stays-literal property that distinguishes
// PathUnescape from QueryUnescape.
func TestParseMetaArgs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want map[string]string
	}{
		{
			name: "empty returns nil",
			args: nil,
			want: nil,
		},
		{
			name: "simple key=value",
			args: []string{"a=b"},
			want: map[string]string{"a": "b"},
		},
		{
			name: "value with encoded space",
			args: []string{"k=%20has%20spaces"},
			want: map[string]string{"k": " has spaces"},
		},
		{
			name: "encoded equals in value via %3D",
			args: []string{"expr=a%3Db"},
			want: map[string]string{"expr": "a=b"},
		},
		{
			name: "encoded equals in key via %3D",
			args: []string{"a%3Db=v"},
			want: map[string]string{"a=b": "v"},
		},
		{
			name: "split on FIRST raw equals only",
			args: []string{"key=value=with=equals"},
			want: map[string]string{"key": "value=with=equals"},
		},
		{
			name: "empty value allowed",
			args: []string{"k="},
			want: map[string]string{"k": ""},
		},
		{
			name: "plus stays literal (PathUnescape, not QueryUnescape)",
			args: []string{"k=a+b"},
			want: map[string]string{"k": "a+b"},
		},
		{
			name: "non-ASCII UTF-8 byte percent-decoded",
			// %E3%81%82 is the UTF-8 encoding of the Japanese character U+3042.
			args: []string{"lang=%E3%81%82"},
			want: map[string]string{"lang": "あ"},
		},
		{
			name: "multiple args merge",
			args: []string{"a=b", "c=d"},
			want: map[string]string{"a": "b", "c": "d"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseMetaArgs(tc.args)
			if err != nil {
				t.Fatalf("ParseMetaArgs(%v) returned unexpected error: %v", tc.args, err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("ParseMetaArgs(%v) = %v, want %v", tc.args, got, tc.want)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("key %q: got %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

// TestParseMetaArgsRejectsNoEquals verifies that an argument with no '=' is
// rejected, and that RunDirect maps that rejection onto a *SetupError.
func TestParseMetaArgsRejectsNoEquals(t *testing.T) {
	if _, err := ParseMetaArgs([]string{"no-equals"}); err == nil {
		t.Fatalf("expected error for argument without '=', got nil")
	}

	// A well-formed arg followed by a malformed one must still fail.
	if _, err := ParseMetaArgs([]string{"a=b", "bad"}); err == nil {
		t.Fatalf("expected error when one argument lacks '=', got nil")
	}
}

// TestParseMetaArgsRejectsBadPercentEscape verifies that an invalid percent
// sequence in either key or value is rejected.
func TestParseMetaArgsRejectsBadPercentEscape(t *testing.T) {
	if _, err := ParseMetaArgs([]string{"k=%ZZ"}); err == nil {
		t.Fatalf("expected error for invalid percent escape in value, got nil")
	}
	if _, err := ParseMetaArgs([]string{"%ZZ=v"}); err == nil {
		t.Fatalf("expected error for invalid percent escape in key, got nil")
	}
}

// TestRunDirectMalformedMetaIsSetupError verifies the malformed --meta path is
// surfaced as a *SetupError (exit code 1) before any output is written.
func TestRunDirectMalformedMetaIsSetupError(t *testing.T) {
	option := NewRecorderOption()
	option.Output = path.Join(t.TempDir(), "should-not-be-created.pgr")
	option.MetaArgs = []string{"missing-equals"}

	err := RunDirect(option)
	if err == nil {
		t.Fatalf("expected error for malformed --meta, got nil")
	}
	var setupErr *SetupError
	if !errors.As(err, &setupErr) {
		t.Fatalf("expected *SetupError, got %T: %v", err, err)
	}
	if _, statErr := os.Stat(option.Output); statErr == nil {
		t.Errorf("output file %q should not have been created on setup error", option.Output)
	}
}
