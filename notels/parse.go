package main

import (
	"bufio"
	"io"
	"os"
)

type File struct {
	AbsPath       string
	Path          string
	OutgoingLinks []Link
	IncomingLinks []Link
}

type Link struct {
	Path string
	// TODO it might be helpful to store both the utf-8 and utf-16 range here
	// because we later cannot convert between the two without having the complete line.
	Range Range
}

// Parses the given files for Obsidian-style markdown links of the form [[link/to/some/file.md]].
func ParseFiles(paths []string, rootPath string) (map[string]*File, error) {
	files := map[string]*File{}
	for _, path := range paths {
		relPath, err := AbsToRelPath(rootPath, path)
		if err != nil {
			continue
		}
		files[relPath] = &File{
			AbsPath: path,
			Path:    relPath,
		}
	}

	for _, f := range files {
		file, err := os.Open(f.AbsPath)
		if err != nil {
			return nil, err
		}
		outgoingLinks, err := ParseLinks(file)
		if err != nil {
			return nil, err
		}
		f.OutgoingLinks = outgoingLinks
		for _, link := range outgoingLinks {
			if g, ok := files[link.Path]; ok {
				nl := Link{
					Path:  f.Path,
					Range: link.Range,
				}
				g.IncomingLinks = append(g.IncomingLinks, nl)
			}
		}
	}

	return files, nil
}

func ParseLinks(r io.Reader) ([]Link, error) {
	scanner := bufio.NewScanner(r)

	var links []Link
	lineIndex := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		lineLinks := ParseLinksInLine(line, lineIndex)
		links = append(links, lineLinks...)
		lineIndex += 1
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return links, nil
}

func ParseLinksInLine(line []byte, lineIndex int) []Link {
	var links []Link
	i := 0
	for i+1 < len(line) {
		// Note: Since [ and ] are ASCII characters we can directly compare the bytes of the
		// utf-8 encoded string.
		if line[i] == '[' && line[i+1] == '[' {
			start := i
			end := -1
			for j := i + 2; j+1 < len(line); j++ {
				if line[j] == ']' && line[j+1] == ']' {
					end = j + 1
					break
				}
			}
			if end == -1 {
				break
			}
			// not an empty link
			if end > start+3 {
				inner := line[start+2 : end-1]
				uStart, uEnd := ToUTF16Range(line, start, end)
				links = append(links, Link{
					Path: string(inner),
					Range: Range{
						Start: Position{
							Line:      uint(lineIndex),
							Character: uint(uStart),
						},
						End: Position{
							Line: uint(lineIndex),
							// the end of an LSP range is exclusive
							Character: uint(uEnd + 1),
						},
					},
				})
			}
			i = end + 1
		} else {
			i += 1
		}
	}
	return links
}

func FindLinkByCharacterOffset(links []Link, offset uint) (Link, bool) {
	for _, link := range links {
		if link.Range.Start.Character <= offset && offset < link.Range.End.Character {
			return link, true
		}
	}
	return Link{}, false
}
