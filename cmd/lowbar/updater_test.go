//go:build windows

package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestParseReleaseVersion(t *testing.T) {
	tests := []struct {
		input string
		want  parsedVersion
	}{
		{input: "0.1.3", want: parsedVersion{0, 1, 3}},
		{input: "v0.1.4", want: parsedVersion{0, 1, 4}},
		{input: "1.2", want: parsedVersion{0, 1, 2}},
		{input: "v1.3", want: parsedVersion{0, 1, 3}},
		{input: "1.0.0", want: parsedVersion{1, 0, 0}},
	}
	for _, tt := range tests {
		got, err := parseReleaseVersion(tt.input)
		if err != nil {
			t.Fatalf("parseReleaseVersion(%q): %v", tt.input, err)
		}
		if got != tt.want {
			t.Fatalf("parseReleaseVersion(%q) = %#v, want %#v", tt.input, got, tt.want)
		}
	}
}

func TestParseReleaseVersionRejectsInvalid(t *testing.T) {
	for _, input := range []string{"", "1", "1.2.3.4", "v1.x.0", "1.-2.0"} {
		if _, err := parseReleaseVersion(input); err == nil {
			t.Fatalf("parseReleaseVersion(%q) unexpectedly succeeded", input)
		}
	}
}

func TestReleaseVersionOrdering(t *testing.T) {
	base := parsedVersion{0, 1, 3}
	if !(parsedVersion{0, 1, 4}).greaterThan(base) {
		t.Fatal("0.1.4 should be newer than 0.1.3")
	}
	if (parsedVersion{0, 1, 3}).greaterThan(base) {
		t.Fatal("same versions must not be newer")
	}
	if (parsedVersion{0, 1, 2}).greaterThan(base) {
		t.Fatal("0.1.2 should not be newer than 0.1.3")
	}
}

func TestValidateReleaseArchive(t *testing.T) {
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "release.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, name := range []string{"LowBar.exe", "LowBarExplorerHook.dll", "Assets/", "Assets/icon.png"} {
		h, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if name != "Assets/" {
			_, _ = h.Write([]byte{1})
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := validateReleaseArchive(zipPath); err != nil {
		t.Fatalf("validateReleaseArchive: %v", err)
	}
}

func TestValidateReleaseArchiveRejectsTraversal(t *testing.T) {
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "bad.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, name := range []string{"LowBar.exe", "LowBarExplorerHook.dll", "Assets/", "../escape.txt"} {
		if _, err := writer.Create(name); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := validateReleaseArchive(zipPath); err == nil {
		t.Fatal("unsafe archive unexpectedly accepted")
	}
}
