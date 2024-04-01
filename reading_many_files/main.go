package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "rf",
		Usage: "measure time to read and parse a lot of files",
		Commands: []*cli.Command{
			{
				Name:  "create-files",
				Usage: "create random files",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "dir",
						Aliases: []string{"d"},
						Value:   "files",
						Usage:   "directory to create files in",
					},
					&cli.IntFlag{
						Name:    "numFiles",
						Aliases: []string{"n"},
						Value:   1000,
						Usage:   "total number of files to create",
					},
					&cli.IntFlag{
						Name:    "numDirs",
						Aliases: []string{"nd"},
						Value:   30,
						Usage:   "number of directories to place files into",
					},
					&cli.IntFlag{
						Name:    "linesPerFile",
						Aliases: []string{"lines"},
						Value:   100,
						Usage:   "number of lines to insert into each file",
					},
					&cli.IntFlag{
						Name:    "linksPerFile",
						Aliases: []string{"links"},
						Value:   10,
						Usage:   "number of links to insert into each file",
					},
				},
				Action: func(cCtx *cli.Context) error {
					dir := cCtx.String("dir")
					numFiles := cCtx.Int("numFiles")
					numDirs := cCtx.Int("numDirs")
					linesPerFile := cCtx.Int("linesPerFile")
					linksPerFile := cCtx.Int("linksPerFile")
					fmt.Printf("Creating %v files with %v lines and %v links each...\n", numFiles, linesPerFile, linksPerFile)
					tStart := time.Now()
					_, err := createFiles(dir, numFiles, numDirs, linesPerFile, linksPerFile)
					timeElapsed := time.Since(tStart)
					fmt.Println("took", timeElapsed)
					return err
				},
			},
			{
				Name:  "read-files",
				Usage: "read and parse random files",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "dir",
						Aliases: []string{"d"},
						Value:   "files",
						Usage:   "directory to read files from",
					},
				},
				Action: func(cCtx *cli.Context) error {
					dir := cCtx.String("dir")
					err := readFiles(dir)
					return err
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

type File struct {
	path  string
	links []string
}

func createFiles(rootPath string, numFiles int, numDirs int, linesPerFile int, linksPerFile int) ([]File, error) {
	if numDirs < 0 {
		numDirs = 0
	}

	dirs := make([]string, numDirs)
	for i := 0; i < numDirs; i++ {
		dirName := fmt.Sprintf("dir%v", i)
		path := filepath.Join(rootPath, dirName)
		dirs[i] = path

		err := os.MkdirAll(path, 0700)
		if err != nil {
			return nil, err
		}
	}

	getRandomDir := func() string {
		if numDirs == 0 {
			return rootPath
		}
		return dirs[rand.Intn(numDirs)]
	}

	files := make([]File, numFiles)
	for i := 0; i < numFiles; i++ {
		dir := getRandomDir()
		fileName := fmt.Sprintf("file%v", i)
		path := filepath.Join(dir, fileName)
		files[i] = File{path: path}
	}

	for i := 0; i < numFiles; i++ {
		for j := 0; j < linksPerFile; j++ {
			otherFile := files[rand.Intn(numFiles)]
			files[i].links = append(files[i].links, otherFile.path)
		}
	}

	for _, f := range files {
		file, err := os.Create(f.path)
		if err != nil {
			return nil, err
		}
		w := bufio.NewWriter(file)
		z := rand.Intn(linesPerFile)
		for i := 0; i < linesPerFile; i++ {
			w.WriteString(sampleText[rand.Intn(len(sampleText))])
			w.WriteString("\n")

			if z == i {
				for _, link := range f.links {
					w.WriteString(genLinkLine(link))
					w.WriteString("\n")
				}
			}
		}
		w.Flush()
	}

	return files, nil
}

func genLinkLine(file string) string {
	before := sampleTextShort[rand.Intn(len(sampleTextShort))]
	after := sampleTextShort[rand.Intn(len(sampleTextShort))]
	link := fmt.Sprintf(" [[%v]] ", file)
	return before + link + after
}

func readFiles(path string) error {
	fmt.Println("Reading files...")
	tStart := time.Now()

	filePaths, err := findFiles(path)
	if err != nil {
		return err
	}

	files, err := parseFiles(filePaths)
	if err != nil {
		fmt.Println(err)
	}

	timeElapsed := time.Since(tStart)
	fmt.Printf("read %v files in %v\n", len(files), timeElapsed)

	for _, f := range files {
		if len(f.links) == 0 {
			fmt.Printf("warning: no links found in file %v", f.path)
		}
	}
	return nil
}

func findFiles(path string) ([]string, error) {
	var filePaths []string
	err := filepath.Walk(path, func(p string, info fs.FileInfo, err error) error {
		if err != nil {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			filePaths = append(filePaths, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return filePaths, nil
}

func parseFiles(filePaths []string) ([]File, error) {
	files := make([]File, len(filePaths))

	for i, f := range filePaths {
		links, err := parseFile(f)
		if err != nil {
			return nil, err
		}
		files[i] = File{path: f, links: links}
	}

	return files, nil
}

// find Obsidian-style Markdown links in a file
// e.g. [[some/path/to/file]]
func parseFile(path string) ([]string, error) {
	r, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var links []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		i := 0
		for i < len(line) {
			// find opening and closing brackets
			if line[i] == '[' {
				if i+1 < len(line) && line[i+1] == '[' {
					start := i + 2
					end := -1
					for j := i + 2; j+1 < len(line); j++ {
						if line[j] == ']' && line[j+1] == ']' {
							end = j
							break
						}
					}
					if end == -1 {
						break
					}
					link := line[start:end]
					if link != "" {
						links = append(links, link)
					}
					i = end + 2
				} else {
					i += 1
				}
			} else {
				i += 1
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return links, nil
}
