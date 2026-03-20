package httputil

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"testing"
)

func TestDecompressReader_Gzip(t *testing.T) {
	want := "hello gzip world"

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.Write([]byte(want))
	gw.Close()

	rc, err := DecompressReader(io.NopCloser(&buf), "http://example.com/playlist.m3u.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDecompressReader_GzipWithQueryParams(t *testing.T) {
	want := "query param test"

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.Write([]byte(want))
	gw.Close()

	rc, err := DecompressReader(io.NopCloser(&buf), "http://example.com/guide.xml.gz?token=abc&foo=bar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDecompressReader_Zip(t *testing.T) {
	want := "hello zip world"

	tmp, err := os.CreateTemp("", "test-*.zip")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	zw := zip.NewWriter(tmp)
	fw, err := zw.Create("playlist.m3u")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write([]byte(want))
	zw.Close()
	tmp.Close()

	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}

	rc, err := DecompressReader(io.NopCloser(bytes.NewReader(data)), "http://example.com/playlist.zip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDecompressReader_Plain(t *testing.T) {
	want := "plain text content"
	body := io.NopCloser(bytes.NewBufferString(want))

	rc, err := DecompressReader(body, "http://example.com/playlist.m3u")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDecompressReader_EmptyZip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	zw.Close()

	_, err := DecompressReader(io.NopCloser(&buf), "http://example.com/empty.zip")
	if err == nil {
		t.Fatal("expected error for empty zip")
	}
	if err.Error() != "zip archive is empty" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDecompressReader_InvalidGzip(t *testing.T) {
	body := io.NopCloser(bytes.NewBufferString("not gzip data"))

	_, err := DecompressReader(body, "http://example.com/bad.gz")
	if err == nil {
		t.Fatal("expected error for invalid gzip")
	}
}
