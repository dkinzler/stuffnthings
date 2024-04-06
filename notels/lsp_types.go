package main

import (
	"encoding/json"
	"fmt"
)

/*
Types defined in the LSP specification.

Note that certain TypeScript constructs can't be represented with a Go struct,
e.g. a property "a?: string | null". We can't distingish between a property not being included
in an object at all or just having a null value, because in Go structs fields always have a zero value.

To represent Typescript optional properties (a?: ...) and null values we can use pointers and json "omitempty" struct tags in Go.
*/

// TODO define all types and also for certain types we have omitted fields

type Message struct {
	Jsonrpc string          `json:"jsonrpc"`
	Id      json.Number     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	Result  json.RawMessage `json:"result"`
	Error   json.RawMessage `json:"error"`
}

type ResponseError struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

func (r ResponseError) Error() string {
	return fmt.Sprintf("code: %v message: %v data: %v", r.Code, r.Message, r.Data)
}

const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603

	ServerNotInitialized = -32002
	UnknownErrorCode     = -32001

	RequestFailed    = -32803
	ServerCancelled  = -32802
	ContentModified  = -32801
	RequestCancelled = -32800
)

type Position struct {
	Line      uint `json:"line"`
	Character uint `json:"character"`
}

const (
	PositionEncodingKindUTF8  = "utf-8"
	PositionEncodingKindUTF16 = "utf-16"
	PositionEncodingKindUTF32 = "utf-32"
)

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type TextDocumentItem struct {
	Uri        string `json:"uri"`
	LangaugeId string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type TextDocumentIdentifier struct {
	Uri string `json:"uri"`
}

type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

type OptionalVersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version *int `json:"version"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type Location struct {
	Uri   string `json:"uri"`
	Range Range  `json:"range"`
}

type LocationLink struct {
	OriginalSelectionRange *Range `json:"originalSelectionRange"`
	TargetUri              string `json:"targetUri"`
	TargetRange            Range  `json:"targetRange"`
	TargetSelectionRange   Range  `json:"targetSelectionRange"`
}

type InitializeParams struct {
	ProcessId             int               `json:"processId"`
	ClientInfo            ClientInfo        `json:"clientInfo"`
	Locale                string            `json:"locale"`
	RootPath              string            `json:"rootPath"`
	RootUri               string            `json:"rootUri"`
	InitializationOptions interface{}       `json:"initializationOptions"`
	Capabilities          Capabilities      `json:"capabilities"`
	Trace                 interface{}       `json:"trace"`
	WorkspaceFolders      []WorkspaceFolder `json:"workspaceFolders"`
}

type Capabilities struct {
	Workspace    interface{} `json:"workspace"`
	TextDocument interface{} `json:"textDocument"`
	Window       interface{} `json:"window"`
	General      interface{} `json:"general"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type WorkspaceFolder struct {
	Uri  string `json:"uri"`
	Name string `json:"name"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   ServerInfo         `json:"serverInfo,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type ServerCapabilities struct {
	PositionEncoding   string                       `json:"positionEncoding,omitempty"`
	TextDocumentSync   *TextDocumentSyncOptions     `json:"textDocumentSync,omitempty"`
	DefinitionProvider bool                         `json:"definitionProvider,omitempty"`
	ReferencesProvider bool                         `json:"referencesProvider,omitempty"`
	Workspace          *WorkspaceServerCapabilities `json:"workspace,omitempty"`
}

type TextDocumentSyncOptions struct {
	OpenClose bool         `json:"openClose"`
	Change    int          `json:"change"`
	Save      *SaveOptions `json:"save,omitempty"`
}

type SaveOptions struct {
	IncludeText bool `json:"includeText"`
}

const (
	TextDocumentSyncKindNone        = 0
	TextDocumentSyncKindFull        = 1
	TextDocumentSyncKindIncremental = 2
)

type WorkspaceServerCapabilities struct {
	FileOperations *FileOperationsServerCapabilities `json:"fileOperations,omitempty"`
}

type FileOperationsServerCapabilities struct {
	DidCreate  *FileOperationRegistrationOptions `json:"didCreate,omitempty"`
	WillCreate *FileOperationRegistrationOptions `json:"willCreate,omitempty"`
	DidRename  *FileOperationRegistrationOptions `json:"didRename,omitempty"`
	WillRename *FileOperationRegistrationOptions `json:"willRename,omitempty"`
	DidDelete  *FileOperationRegistrationOptions `json:"didDelete,omitempty"`
	WillDelete *FileOperationRegistrationOptions `json:"willDelete,omitempty"`
}

type FileOperationRegistrationOptions struct {
	Filter []FileOperationFilter `json:"filter"`
}

type FileOperationFilter struct {
	Scheme  string               `json:"scheme,omitempty"`
	Pattern FileOperationPattern `json:"pattern"`
}

type FileOperationPattern struct {
	Glob    string      `json:"glob"`
	Matches interface{} `json:"matches,omitempty"`
	Options interface{} `json:"options,omitempty"`
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier   `json:"textDocument"`
	ContentChanges []TextDocumentContentChangedEvent `json:"contentChanges"`
}

// If server uses TextDocumentSyncKindFull the range will be empty
// and text will contain the new text of the whole document.
type TextDocumentContentChangedEvent struct {
	Range Range  `json:"range"`
	Text  string `json:"text"`
}

type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         *string                `json:"text"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type DefinitionParams struct {
	TextDocumentPositionParams
}

type ReferenceParams struct {
	TextDocumentPositionParams
}
