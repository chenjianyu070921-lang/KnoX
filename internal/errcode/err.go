package errcode

import "fmt"

// BizError 业务错误，包含业务码和错误描述
type BizError struct {
	Code    int
	Message string
}

func (e *BizError) Error() string {
	return fmt.Sprintf("code=%d, message=%s", e.Code, e.Message)
}

func New(code int, msg ...string) *BizError {
	m := Msg(code)
	if len(msg) > 0 {
		m = msg[0]
	}
	return &BizError{Code: code, Message: m}
}
