package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"unicode/utf16"
)

// Returns the path of a URI.
// "file:///a/b/c" -> "/a/b/c"
func URItoPath(uri string) (string, error) {
	url, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	path := filepath.FromSlash(url.Path)
	return path, nil
}

// Returns a file URI for the given path.
// "/a/b/c" -> "file:///a/b/c"
func PathToURI(path string) string {
	path = filepath.ToSlash(path)
	return fmt.Sprintf("file://%v", path)
}

// Converts absolute to relative path.
// AbsToRelPath("/a/b", "/a/b/c/d") -> "c/d"
func AbsToRelPath(prefix, path string) (string, error) {
	relPath, err := filepath.Rel(prefix, path)
	if err != nil {
		return "", err
	}
	return relPath, nil
}

// Converts relative to absolute path.
// RelToAbsPath("/a/b", "c/d") -> "/a/b/c/d"
func RelToAbsPath(prefix, path string) string {
	return filepath.Join(prefix, path)
}

func NewFileScanner(filePath string) (*bufio.Scanner, error) {
	r, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	return bufio.NewScanner(r), nil
}

// Reads and returns a specific line from a file.
func ReadLine(filePath string, index int) ([]byte, error) {
	scanner, err := NewFileScanner(filePath)
	if err != nil {
		return nil, err
	}

	var line []byte
	li := 0
	for scanner.Scan() {
		if li == index {
			line = scanner.Bytes()
			break
		}
		li += 1
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return line, nil
}

// Returns a list of all files contained (recursively) in the given directory.
func ListFiles(path string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}
		if !d.IsDir() {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// Converts a utf-8 range of code units (bytes) to the equivalent utf-16 range of code units (2 bytes each).
// TODO this could probably be done more efficiently?
func ToUTF16Range(s string, start, end int) (int, int) {
	a := utf16.Encode([]rune(s[0:start]))
	b := utf16.Encode([]rune(s[start : end+1]))
	return len(a), len(a) + len(b) - 1
}
