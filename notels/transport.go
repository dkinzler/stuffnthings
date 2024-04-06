package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Reader struct {
	r *bufio.Reader
}

func NewReader(r io.Reader) Reader {
	return Reader{
		r: bufio.NewReaderSize(r, 1<<16),
	}
}

func (mr Reader) ReadMessage() ([]byte, string, error) {
	contentLength, contentType, err := mr.readHeader()
	if err != nil {
		return nil, "", err
	}
	content, err := mr.readContent(contentLength)
	if err != nil {
		return nil, "", err
	}
	return content, contentType, nil
}

const defaultContentType = "application/vscode-jsonrpc; charset=utf-8"

// TODO could probably make this more robust
func (mr Reader) readHeader() (int, string, error) {
	contentLength := -1
	contentType := defaultContentType
	for {
		field, err := mr.r.ReadString('\n')
		if err != nil {
			return 0, "", err
		}
		if field == "\r\n" {
			break
		}
		name, value, err := parseHeaderField(field)
		if err != nil {
			return 0, "", err
		}
		if name == "Content-Length" {
			n, err := strconv.Atoi(value)
			if err != nil {
				return 0, "", err
			}
			contentLength = n
		} else if name == "Content-Type" {
			// TODO implement parsing content-type field
			// only accept value right now is application/vscode-jsonrpc; charset=utf-8
		} else {
			return 0, "", errors.New("MessageReader: unknown header field")
		}
	}
	if contentLength == -1 {
		return 0, "", errors.New("MessageReader: Content-Length header field missing")
	}
	return contentLength, contentType, nil
}

func parseHeaderField(field string) (string, string, error) {
	name, value, ok := strings.Cut(field, ": ")
	if !ok {
		return "", "", errors.New("MessageReader: invalid header field")
	}
	value = strings.TrimSuffix(value, "\r\n")
	return name, value, nil
}

func (mr Reader) readContent(length int) ([]byte, error) {
	buf := make([]byte, length)
	_, err := io.ReadFull(mr.r, buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

type Writer struct {
	w *bufio.Writer
}

func NewWriter(w io.Writer) Writer {
	return Writer{
		w: bufio.NewWriter(w),
	}
}

func (mw Writer) Write(content []byte) error {
	n := len(content)
	_, err := mw.w.WriteString(fmt.Sprintf("Content-Length: %v\r\n", n))
	if err != nil {
		return err
	}
	_, err = mw.w.WriteString("Content-Type: application/vscode-jsonrpc; charset=utf-8\r\n\r\n")
	if err != nil {
		return err
	}
	_, err = mw.w.Write(content)
	if err != nil {
		return err
	}
	err = mw.w.Flush()
	if err != nil {
		return err
	}
	return nil
}

func IsEOFError(err error) bool {
	return err == io.EOF
}
