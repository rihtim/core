package messages

import (
	"io"
	"mime/multipart"
)

type Message struct {
	Rid           int `json:"rid,omitempty"`
	Res           string `json:"res,omitempty"`
	Command       string `json:"cmd,omitempty"`
	Headers       map[string][]string `json:"headers,omitempty"`
	Parameters    map[string][]string `json:"parameters,omitempty"`
	MultipartForm *multipart.Form `json:"multipart,omitempty"`
	Body          map[string]interface{} `json:"body,omitempty"`
	RawBody       []byte `json:"rawbody,omitempty"`	// used for files
	ReqBodyRaw    io.ReadCloser
	Status        int `json:"status,omitempty"` 	// used only in responses
}

type RequestWrapper struct {
	Message  Message
	Listener chan Message
}

type RequestError struct {
	Code    int
	Message string
	Body    map[string]interface{}
}
