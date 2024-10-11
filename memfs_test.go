package memfs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
)

func TestFS(t *testing.T) {
	rootFS := New()

	err := rootFS.MkdirAll("foo/bar", 0o777)
	if err != nil {
		t.Fatal(err)
	}
	err = rootFS.WriteFile("foo/bar/buz.txt", []byte("buz"), 0o777)
	if err != nil {
		t.Fatal(err)
	}
	err = fstest.TestFS(rootFS, "foo/bar/buz.txt")
	if err != nil {
		t.Fatal(err)
	}
}

func TestMemFS(t *testing.T) {
	rootFS := New()

	err := rootFS.MkdirAll("foo/bar", 0o777)
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

	err = rootFS.WriteFile("foo/baz/buz.txt", []byte("buz"), 0o777)
	if err == nil && errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Expected missing directory error but got none")
	}

	_, err = fs.ReadFile(rootFS, "foo/baz/buz.txt")
	if err == nil && errors.Is(err, fs.ErrNotExist) {
		t.Fatal("Expected no such file but got no error")
	}

	body := []byte("baz")
	err = rootFS.WriteFile("foo/bar/baz.txt", body, 0o777)
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

	subFS, err := rootFS.Sub("foo/bar")
	if err != nil {
		t.Fatal(err)
	}

	gotSubBody, err := fs.ReadFile(subFS, "baz.txt")
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(body, gotSubBody); diff != "" {
		t.Fatalf("write/read baz.txt mismatch %s", diff)
	}

	body = []byte("top_level_file")
	err = rootFS.WriteFile("top_level_file.txt", body, 0o777)
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

func TestOpenHook(t *testing.T) {
	openHook := func(path string, content []byte, origError error) ([]byte, error) {
		if path == "foo/bar/override" {
			return []byte("overriden content"), nil
		}

		return content, origError
	}

	rootFS := New(WithOpenHook(openHook))

	err := rootFS.MkdirAll("foo/bar", 0o777)
	if err != nil {
		t.Fatal(err)
	}

	rootFS.WriteFile("foo/bar/f1", []byte("f1"), 0o777)
	rootFS.WriteFile("foo/bar/override", []byte("orig content"), 0o777)

	content, err := fs.ReadFile(rootFS, "foo/bar/f1")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(content), "f1"); diff != "" {
		t.Fatalf("write/read roo/bar/f1 mismatch %s", diff)
	}

	content, err = fs.ReadFile(rootFS, "foo/bar/override")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(content), "overriden content"); diff != "" {
		t.Fatalf("hook read mismatch %s", diff)
	}

	_, err = fs.ReadFile(rootFS, "foo/bar/non_existing_file")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Expected ErrNotExist for non-existing file, got: %v", err)
	}
}

func TestSeek(t *testing.T) {
	rootFS := New()

	err := rootFS.WriteFile("foo", []byte("0123456789"), 0o777)
	if err != nil {
		t.Fatal(err)
	}

	f, err := rootFS.Open("foo")
	if err != nil {
		t.Fatal(err)
	}

	seeker, ok := f.(io.Seeker)
	if !ok {
		t.Fatalf("File does not implement io.Seeker")
	}

	// Read first bytes.
	bs := make([]byte, 3)
	n, err := f.Read(bs)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("Expected 3 bytes read, got %d", n)
	}
	if diff := cmp.Diff(bs, []byte("012")); diff != "" {
		t.Fatalf("read mismatch %s", diff)
	}

	// Read more bytes, make sure reader tracks.
	n, err = f.Read(bs)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("Expected 3 bytes read, got %d", n)
	}
	if diff := cmp.Diff(bs, []byte("345")); diff != "" {
		t.Fatalf("read mismatch %s", diff)
	}

	// Seek to beginning.
	ofs, err := seeker.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}
	if ofs != 0 {
		t.Fatalf("Expected offset 0, got %d", ofs)
	}

	// Read first bytes again.
	n, err = f.Read(bs)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("Expected 3 bytes read, got %d", n)
	}
	if diff := cmp.Diff(bs, []byte("012")); diff != "" {
		t.Fatalf("read mismatch %s", diff)
	}
}
