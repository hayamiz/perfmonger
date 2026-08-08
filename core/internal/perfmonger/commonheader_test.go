package perfmonger

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"testing"
	"time"
)

// TestCommonHeaderMetaGobRoundTrip verifies that CommonHeader.Meta survives a
// gob encode -> decode cycle intact, including keys/values that contain spaces,
// '=' and non-ASCII characters (i.e. the decoded forms produced by --meta).
func TestCommonHeaderMetaGobRoundTrip(t *testing.T) {
	orig := CommonHeader{
		Platform:  Linux,
		Hostname:  "testhost",
		StartTime: time.Unix(1700000000, 0),
		Meta: map[string]string{
			"a":    "b",
			"k":    " has spaces=",
			"lang": "あ",
		},
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(&orig); err != nil {
		t.Fatalf("gob encode failed: %v", err)
	}

	var decoded CommonHeader
	if err := gob.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatalf("gob decode failed: %v", err)
	}

	if !reflect.DeepEqual(decoded.Meta, orig.Meta) {
		t.Errorf("Meta round-trip mismatch: got %v, want %v", decoded.Meta, orig.Meta)
	}
}

// TestCommonHeaderNilMetaGobRoundTrip verifies backward compatibility: a header
// written without any metadata decodes with a nil/empty Meta map, and readers
// can treat it as "no metadata".
func TestCommonHeaderNilMetaGobRoundTrip(t *testing.T) {
	orig := CommonHeader{Platform: Linux, Hostname: "h", StartTime: time.Unix(1, 0)}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(&orig); err != nil {
		t.Fatalf("gob encode failed: %v", err)
	}

	var decoded CommonHeader
	if err := gob.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatalf("gob decode failed: %v", err)
	}

	if len(decoded.Meta) != 0 {
		t.Errorf("expected empty Meta, got %v", decoded.Meta)
	}
	if len(decoded.SortedMetaKeys()) != 0 {
		t.Errorf("expected no sorted meta keys, got %v", decoded.SortedMetaKeys())
	}
}

// TestSortedMetaKeys verifies keys are returned in ascending order.
func TestSortedMetaKeys(t *testing.T) {
	h := CommonHeader{Meta: map[string]string{"c": "3", "a": "1", "b": "2"}}
	got := h.SortedMetaKeys()
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SortedMetaKeys() = %v, want %v", got, want)
	}
}
