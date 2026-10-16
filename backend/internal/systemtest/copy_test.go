package systemtest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyPathCopiesFileIntoDirectory(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	destination := filepath.Join(root, "destination")
	if err := os.WriteFile(source, []byte("fntube"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destination, 0755); err != nil {
		t.Fatal(err)
	}

	result, err := CopyPath(source, destination)
	if err != nil {
		t.Fatalf("CopyPath returned error: %v", err)
	}
	if result.Files != 1 || result.Bytes != 6 {
		t.Fatalf("unexpected result: %+v", result)
	}
	content, err := os.ReadFile(filepath.Join(destination, "source.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "fntube" {
		t.Fatalf("unexpected copied content: %q", content)
	}
}

func TestCopyPathCopiesDirectoryTree(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "test.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := CopyPath(source, filepath.Join(root, "copied"))
	if err != nil {
		t.Fatalf("CopyPath returned error: %v", err)
	}
	if result.Files != 1 || result.Directories != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "copied", "nested", "test.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestCopyPathRejectsNestedDestination(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}

	if _, err := CopyPath(source, filepath.Join(source, "copy")); err == nil {
		t.Fatal("expected nested destination to be rejected")
	}
}

func TestCopyPathDoesNotOverwrite(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	destination := filepath.Join(root, "destination.txt")
	if err := os.WriteFile(source, []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("destination"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := CopyPath(source, destination); err == nil {
		t.Fatal("expected existing destination to be rejected")
	}
}
