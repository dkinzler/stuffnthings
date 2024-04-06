package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "notels",
		Usage: "Note Language Server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "logFile",
				Aliases: []string{"l"},
				Value:   "",
				Usage:   "save log messages to file",
			},
		},
		Action: func(cCtx *cli.Context) error {
			logFile := cCtx.String("logFile")
			err := setupLogging(logFile)
			if err != nil {
				return err
			}
			r := NewReader(os.Stdin)
			w := NewWriter(os.Stdout)
			srv := NewServer()
			return run(r, w, srv)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func setupLogging(logFile string) error {
	if logFile == "" {
		return nil
	}
	f, err := os.Create(logFile)
	if err != nil {
		return err
	}
	log.SetOutput(io.MultiWriter(os.Stdout, f))
	return nil
}

func run(r Reader, w Writer, srv *Server) error {
	for {
		content, _, err := r.ReadMessage()
		if err != nil {
			if IsEOFError(err) {
				return nil
			}
			log.Println(err)
			return err
		}

		var m Message
		err = json.Unmarshal(content, &m)
		if err != nil {
			log.Println(err)
			response := encodeParseErrorResponse()
			err := sendResponse(response, w)
			if err != nil {
				log.Println(err)
			}
			continue
		}

		if m.Jsonrpc != "2.0" {
			response := encodeErrorResponse(InvalidRequest, "protocol version is not 2.0", nil)
			err := sendResponse(response, w)
			if err != nil {
				log.Println(err)
			}
			continue
		}

		var response map[string]interface{}

		log.Printf("got message with method \"%v\"", m.Method)

		switch m.Method {
		case "initialize":
			params, err := decodeParams[InitializeParams](m.Params)
			if err != nil {
				log.Println(err)
				response = encodeParseErrorResponse()
				break
			}
			result, err := srv.Initialize(params)
			response = encodeResponse(m.Id, result, err)
		case "initialized":
			srv.HandleInitialized()
		case "shutdown":
			err := srv.Shutdown()
			response = encodeResponse(m.Id, nil, err)
		case "exit":
			ok := srv.HandleExit()
			if !ok {
				return errors.New("received unexpected exit notification")
			}
			return nil
		case "textDocument/didOpen":
			params, err := decodeParams[DidOpenTextDocumentParams](m.Params)
			if err != nil {
				log.Println(err)
				response = encodeParseErrorResponse()
				break
			}
			srv.HandleTextDocumentDidOpen(params)
		case "textDocument/didChange":
			params, err := decodeParams[DidChangeTextDocumentParams](m.Params)
			if err != nil {
				log.Println(err)
				response = encodeParseErrorResponse()
				break
			}
			srv.HandleTextDocumentDidChange(params)
		case "textDocument/didSave":
			params, err := decodeParams[DidSaveTextDocumentParams](m.Params)
			if err != nil {
				log.Println(err)
				response = encodeParseErrorResponse()
				break
			}
			srv.HandleTextDocumentDidSave(params)
		case "textDocument/didClose":
			params, err := decodeParams[DidCloseTextDocumentParams](m.Params)
			if err != nil {
				log.Println(err)
				response = encodeParseErrorResponse()
				break
			}
			srv.HandleTextDocumentDidClose(params)
		case "textDocument/definition":
			params, err := decodeParams[DefinitionParams](m.Params)
			if err != nil {
				log.Println(err)
				response = encodeParseErrorResponse()
				break
			}
			result, err := srv.GotoDefinition(params)
			response = encodeResponse(m.Id, result, err)
		case "textDocument/references":
			params, err := decodeParams[ReferenceParams](m.Params)
			if err != nil {
				log.Println(err)
				response = encodeParseErrorResponse()
				break
			}
			result, err := srv.FindReferences(params)
			response = encodeResponse(m.Id, result, err)
		default:
			log.Println("error unknown method:", m.Method)
			response = encodeResponse(m.Id, nil, ResponseError{
				Code:    MethodNotFound,
				Message: "method not found",
			})
		}

		if response != nil {
			err := sendResponse(response, w)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func decodeParams[V any](x json.RawMessage) (V, error) {
	var v V
	err := json.Unmarshal(x, &v)
	return v, err
}

func encodeResponse(id json.Number, result interface{}, err error) map[string]interface{} {
	var response map[string]interface{}
	var idValue interface{}
	if id == "" {
		idValue = nil
	} else {
		idValue = id
	}
	if err != nil {
		response = map[string]interface{}{
			"id":    idValue,
			"error": err,
		}
	} else {
		response = map[string]interface{}{
			"id":     idValue,
			"result": result,
		}
	}
	return response
}

func encodeErrorResponse(code int, message string, data map[string]interface{}) map[string]interface{} {
	return encodeResponse("", nil, ResponseError{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

func encodeParseErrorResponse() map[string]interface{} {
	return encodeErrorResponse(ParseError, "error parsing message", nil)
}

func sendResponse(response map[string]interface{}, w Writer) error {
	r, err := json.Marshal(response)
	if err != nil {
		return err
	}
	err = w.Write(r)
	if err != nil {
		return err
	}
	return nil
}
