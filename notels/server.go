package main

import (
	"log"
)

const (
	ServerStateNotInitialized    = 0
	ServerStateInitialized       = 1
	ServerStateShutdownRequested = 2
)

// Methods that process notification begin with "Handle".
// Methods that process a request should return a result and an error that is either nil
// or of type ResponseError.
//
// Note the default encoding used in the LSP is utf-16.
// Positions are defined as line number + character offset pairs,
// where the character offset is the (0-based) index of the first utf-16 code unit
// that belongs to the character.
// In this LSP implementation we internally use utf-8 to read and process text.
// Incoming positions/ranges from the client are converted to utf-8 positions/ranges.
// Outgoing positions/ranges are converted to utf-16.
// This trades-off (possibly) some performance for ease of implementation.
type Server struct {
	state    int
	rootPath string
	// TODO to keep this data consistent we would to reparse files every time they get modified
	// listen to file rename, creation, deletion events etc.
	files map[string]*File
}

func NewServer() *Server {
	return &Server{
		state: ServerStateNotInitialized,
	}
}

func (s *Server) isReady() error {
	if s.state == ServerStateInitialized {
		return nil
	} else if s.state == ServerStateNotInitialized {
		return ResponseError{
			Code:    ServerNotInitialized,
			Message: "server not initialized",
		}
	} else {
		return ResponseError{
			Code:    InvalidRequest,
			Message: "invalid request, server has shutdown",
		}
	}
}

func (s *Server) Initialize(params InitializeParams) (InitializeResult, error) {
	if s.state != ServerStateNotInitialized {
		return InitializeResult{}, ResponseError{
			Code:    RequestFailed,
			Message: "server is already initialized",
		}
	}

	err := s.readWorkspace(params.RootUri)
	if err != nil {
		log.Println("error reading root dir:", err)
		return InitializeResult{}, ResponseError{
			Code:    InternalError,
			Message: "error reading root directory",
		}
	}

	capabilities := ServerCapabilities{
		PositionEncoding: PositionEncodingKindUTF16,
		TextDocumentSync: &TextDocumentSyncOptions{
			OpenClose: true,
			Change:    TextDocumentSyncKindFull,
			Save: &SaveOptions{
				IncludeText: true,
			},
		},
		DefinitionProvider: true,
		ReferencesProvider: true,
	}

	result := InitializeResult{
		Capabilities: capabilities,
		ServerInfo: ServerInfo{
			Name:    "NoteLS",
			Version: "0.0.1",
		},
	}

	s.state = ServerStateInitialized

	return result, nil
}

func (s *Server) readWorkspace(rootUri string) error {
	rootPath, err := URItoPath(rootUri)
	if err != nil {
		return err
	}

	filePaths, err := ListFiles(rootPath)
	if err != nil {
		return err
	}

	files, err := ParseFiles(filePaths, rootPath)
	if err != nil {
		return err
	}

	s.rootPath = rootPath
	s.files = files
	return nil
}

func (s *Server) HandleInitialized() {
	if s.isReady() != nil {
		return
	}
}

func (s *Server) Shutdown() error {
	if err := s.isReady(); err != nil {
		return err
	}
	s.state = ServerStateShutdownRequested
	return nil
}
func (s *Server) HandleExit() bool {
	return s.state == ServerStateShutdownRequested
}

func (s *Server) HandleTextDocumentDidOpen(params DidOpenTextDocumentParams) {
	if s.isReady() != nil {
		return
	}
}

func (s *Server) HandleTextDocumentDidChange(params DidChangeTextDocumentParams) {
	if s.isReady() != nil {
		return
	}
}

func (s *Server) HandleTextDocumentDidSave(params DidSaveTextDocumentParams) {
	if s.isReady() != nil {
		return
	}
}

func (s *Server) HandleTextDocumentDidClose(params DidCloseTextDocumentParams) {
	if s.isReady() != nil {
		return
	}
}

// Returns the location for the link at the given position.
// Right now links need to be in the format [[relative/path/to/file.md]] but it would not be hard
// to support more advanced formats like [[link|displayText]] or [[link#section]] etc.
func (s *Server) GotoDefinition(params DefinitionParams) (Location, error) {
	if err := s.isReady(); err != nil {
		return Location{}, err
	}

	path, err := URItoPath(params.TextDocument.Uri)
	if err != nil {
		return Location{}, ResponseError{
			Code:    InvalidRequest,
			Message: "invalid uri",
		}
	}

	line, err := ReadLine(path, int(params.Position.Line))
	if err != nil {
		return Location{}, ResponseError{
			Code:    InternalError,
			Message: "could not read file",
		}
	}

	links, err := ParseLinksInLine(string(line), 0)
	if err != nil {
		return Location{}, ResponseError{
			Code:    InternalError,
			Message: "could not parse link",
		}
	}

	link, ok := FindLinkByCharacterOffset(links, params.Position.Character)
	if !ok {
		return Location{}, ResponseError{
			Code:    InternalError,
			Message: "no link found",
		}
	}

	for path := range s.files {
		if path == link.Path {
			return Location{
				Uri: PathToURI(RelToAbsPath(s.rootPath, path)),
				Range: Range{
					Start: Position{
						Line:      0,
						Character: 0,
					},
					End: Position{
						Line:      1,
						Character: 0,
					},
				},
			}, nil
		}
	}

	return Location{}, ResponseError{
		Code:    InternalError,
		Message: "file not found",
	}
}

func (s *Server) FindReferences(params ReferenceParams) ([]Location, error) {
	if err := s.isReady(); err != nil {
		return nil, err
	}

	path, err := URItoPath(params.TextDocument.Uri)
	if err != nil {
		return nil, ResponseError{
			Code:    InvalidRequest,
			Message: "invalid uri",
		}
	}
	path, err = AbsToRelPath(s.rootPath, path)
	if err != nil {
		return nil, ResponseError{
			Code:    InvalidRequest,
			Message: "file not found",
		}
	}

	file, ok := s.files[path]
	if !ok {
		return nil, ResponseError{
			Code:    InvalidRequest,
			Message: "file not found",
		}
	}

	var result []Location
	for _, link := range file.IncomingLinks {
		result = append(result, Location{
			Uri:   PathToURI(RelToAbsPath(s.rootPath, link.Path)),
			Range: link.Range,
		})
	}

	return result, nil
}
