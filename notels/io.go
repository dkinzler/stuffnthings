package main

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"unicode/utf16"
	"unicode/utf8"
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

func ReadLineFromFile(path string, index int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return ReadLine(file, index)
}

// Reads and returns a specific line from a file.
func ReadLine(r io.Reader, index int) ([]byte, error) {
	scanner := bufio.NewScanner(r)

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

// TODO this could probably be done more efficiently?
// Converts a utf-8 range of code units (bytes) to the equivalent utf-16 range of code units (2 bytes each).
func ToUTF16Range(s []byte, start, end int) (int, int) {
	a := countUTF16Units(s[0:start])
	b := countUTF16Units(s[start : end+1])
	return a, a + b - 1
}

func countUTF16Units(s []byte) int {
	result := 0
	for len(s) > 0 {
		r, size := utf8.DecodeRune(s)
		a, b := utf16.EncodeRune(r)
		if a == b && a == 0xfffd {
			result += 1
		} else {
			result += 2
		}
		s = s[size:]
	}
	return result
}
