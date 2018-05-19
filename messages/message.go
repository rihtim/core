package messages

import (
	"io"
	"strconv"
	"mime/multipart"
	"github.com/rihtim/core/utils"
)

type Message struct {
	Rid           int                    `json:"rid,omitempty"`
	IP            string                 `json:"ip,omitempty"`
	Res           string                 `json:"res,omitempty"`
	Command       string                 `json:"method,omitempty"`
	Headers       map[string][]string    `json:"headers,omitempty"`
	Parameters    map[string][]string    `json:"parameters,omitempty"`
	MultipartForm *multipart.Form        `json:"multipart,omitempty"`
	Body          map[string]interface{} `json:"body,omitempty"`
	RawBody       []byte                 `json:"rawbody,omitempty"` // used for files
	ReqBodyRaw    io.ReadCloser
	Status        int                    `json:"status,omitempty"` // used only in responses
}

func (m Message) GetParameter(key string) (value string, contains bool) {
	if m.Parameters == nil {
		contains = false
		return
	}

	values, contains := m.Parameters[key]
	if contains {
		value = values[0]
	}
	return
}

func (m Message) GetIntParameter(key string) (valueAsNumber int, contains bool, err *utils.Error) {
	if m.Parameters == nil {
		contains = false
		return
	}

	values, contains := m.Parameters[key]
	if contains {
		number, convertErr := strconv.Atoi(values[0])
		if convertErr != nil {
			err = &utils.Error{
			    Message: convertErr.Error(),
			}
			return
		}
		valueAsNumber = number
	}
	return
}

func (m Message) GetHeader(key string) (value string, contains bool) {
	if m.Headers == nil {
		contains = false
		return
	}

	values, contains := m.Headers[key]
	if contains {
		value = values[0]
	}
	return
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

func (m *Message) IsEmpty() bool {
	return m.Status == 0 && len(m.Res) == 0 && len(m.Command) == 0 && m.Headers == nil && m.Parameters == nil && m.MultipartForm == nil && m.Body == nil && len(m.RawBody) == 0 && m.ReqBodyRaw == nil
}
