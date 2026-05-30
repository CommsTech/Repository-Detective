package gitea

import (
	"testing"
)

func TestDecodeRepositoryContentsDirectoryArray(t *testing.T) {
	body := []byte(`[
		{"name":"main.go","path":"main.go","sha":"abc","size":100,"type":"file"},
		{"name":"README.md","path":"README.md","sha":"def","size":200,"type":"file"}
	]`)

	items, err := decodeRepositoryContents(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Name != "main.go" {
		t.Fatalf("expected main.go, got %s", items[0].Name)
	}
}

func TestDecodeRepositoryContentsSingleFile(t *testing.T) {
	body := []byte(`{"name":"main.go","path":"main.go","sha":"abc","size":100,"type":"file","content":"aGVsbG8="}`)

	items, err := decodeRepositoryContents(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Type != "file" {
		t.Fatalf("expected file type, got %s", items[0].Type)
	}
}

func TestDecodeRepositoryContentsEmptyArray(t *testing.T) {
	items, err := decodeRepositoryContents([]byte(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty slice, got %d items", len(items))
	}
}
