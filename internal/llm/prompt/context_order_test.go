package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContextOverlappingPathsPreserveOrder(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a", "b"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("content-"+name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	got := processContextPaths(dir, []string{"b", "./", "a"})
	if strings.Count(got, "content-b") != 1 || strings.Count(got, "content-a") != 1 || strings.Index(got, "content-b") > strings.Index(got, "content-a") {
		t.Fatalf("unexpected order or duplicates: %s", got)
	}
}
