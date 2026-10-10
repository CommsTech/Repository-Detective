package gitea

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestErrNotDirectoryWrapsPath(t *testing.T) {
	err := fmt.Errorf("%w: %s", errNotDirectory, ".gitea/workflows")
	if !errors.Is(err, errNotDirectory) {
		t.Fatal("expected errNotDirectory in chain")
	}
	if !strings.Contains(err.Error(), "path is not a directory") {
		t.Fatalf("unexpected message: %v", err)
	}
}
