package memfs

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMemFS(t *testing.T) {

	rootFS := New()

	err := rootFS.MkdirAll("foo/bar", 0777)
	if err != nil {
		t.Fatal(err)
	}

	var gotPaths []string

	err = fs.WalkDir(rootFS, ".", func(path string, d fs.DirEntry, err error) error {
		gotPaths = append(gotPaths, path)
		if !d.IsDir() {
			return fmt.Errorf("%s is not a directory", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	expectPaths := []string{
		".",
		"foo",
		"foo/bar",
	}

	if diff := cmp.Diff(expectPaths, gotPaths); diff != "" {
		t.Fatalf("WalkDir mismatch %s", diff)
	}

	err = rootFS.WriteFile("foo/baz/buz.txt", []byte("buz"), 0777)
	if err == nil && errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Expected missing directory error but got none")
	}

	_, err = fs.ReadFile(rootFS, "foo/baz/buz.txt")
	if err == nil && errors.Is(err, fs.ErrNotExist) {
		t.Fatal("Expected no such file but got no error")
	}

	body := []byte("baz")
	err = rootFS.WriteFile("foo/bar/baz.txt", body, 0777)
	if err != nil {
		t.Fatal(err)
	}

	gotBody, err := fs.ReadFile(rootFS, "foo/bar/baz.txt")
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(body, gotBody); diff != "" {
		t.Fatalf("write/read baz.txt mismatch %s", diff)
	}

	body = []byte("top_level_file")
	err = rootFS.WriteFile("top_level_file.txt", body, 0777)
	if err != nil {
		t.Fatalf("Write top_level_file error: %s", err)
	}

	gotBody, err = fs.ReadFile(rootFS, "top_level_file.txt")
	if err != nil {
		t.Fatalf("Read top_level_file error: %s", err)
	}

	if diff := cmp.Diff(body, gotBody); diff != "" {
		t.Fatalf("write/read top_level_file.txt mismatch %s", diff)
	}
}

func TestMemFSremove(t *testing.T) {

	rootFS := New()
	err := rootFS.MkdirAll("foo/bar/baz", 0777)
	if err != nil {
		t.Fatal(err)
	}

	var gotPaths []string
	err = fs.WalkDir(rootFS, ".", func(path string, d fs.DirEntry, err error) error {
		gotPaths = append(gotPaths, path)
		if !d.IsDir() {
			return fmt.Errorf("%s is not a directory", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	expectPaths := []string{
		".",
		"foo",
		"foo/bar",
		"foo/bar/baz",
	}

	if diff := cmp.Diff(expectPaths, gotPaths); diff != "" {
		t.Fatalf("WalkDir mismatch %s", diff)
	}

	err = rootFS.WriteFile("foo/bar/baz/buz.txt", []byte("buz"), 0777)
	if err == nil && errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Expected missing directory error but got none")
	}

	//fire a delete that WILL faila
	if err := rootFS.Remove("foo/bar/baz/buzz"); err == nil {
		t.Fatalf("Failed to catch non-existent foo/bar/baz/buzz")
	}

	//fire one that will succeed
	if err := rootFS.Remove("foo/bar/baz"); err != nil {
		t.Fatalf("Failed to delete foo/bar/baz: %v", err)
	}

	//walk it again
	gotPaths = nil
	err = fs.WalkDir(rootFS, ".", func(path string, d fs.DirEntry, err error) error {
		gotPaths = append(gotPaths, path)
		if !d.IsDir() {
			return fmt.Errorf("%s is not a directory", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// check status
	expectPaths = expectPaths[0:3]
	if diff := cmp.Diff(expectPaths, gotPaths); diff != "" {
		t.Fatalf("WalkDir mismatch %s", diff)
	}
}
