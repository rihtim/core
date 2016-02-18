package utils

import "fmt"

type Error struct {
	Code    int `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("Error: %d - %s", e.Code, e.Message)
}
