package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hktkhub/commonsdk/http/codec"
)

// envelope 是后端统一响应结构。
type envelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// Result 是一次 HTTP 调用的完整结果。
// HTTPStatus 始终是原始 HTTP 状态码，Code/Msg 来自业务响应体。
type Result struct {
	Code       int
	Msg        string
	Data       json.RawMessage
	HTTPStatus int
	Header     http.Header

	codec codec.Codec
}

// Decode 将 Data 解码到 v。Data 为空或 null 时不做处理。
func (r *Result) Decode(v any) error {
	if r == nil {
		return errors.New("http/client: nil result")
	}
	data := bytes.TrimSpace(r.Data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil
	}
	cdc := r.codec
	if cdc == nil {
		cdc = codec.Default
	}
	return cdc.Unmarshal(data, v)
}

// APIError 表示业务响应或 HTTP 层的非成功结果。
// 有统一响应体时 Code/Msg 来自响应体；否则 Code 为 0，Msg 为 HTTP 状态摘要。
type APIError struct {
	Code       int
	Msg        string
	HTTPStatus int
	Body       []byte
}

func (e *APIError) Error() string {
	msg := strings.TrimSpace(e.Msg)
	if msg == "" && len(e.Body) > 0 {
		msg = strings.TrimSpace(string(e.Body))
	}

	switch {
	case e.Code != 0 && msg != "":
		return fmt.Sprintf("http/client: code=%d msg=%s", e.Code, msg)
	case e.Code != 0:
		return fmt.Sprintf("http/client: code=%d http=%d", e.Code, e.HTTPStatus)
	case msg != "":
		return fmt.Sprintf("http/client: http=%d msg=%s", e.HTTPStatus, msg)
	default:
		return fmt.Sprintf("http/client: http=%d", e.HTTPStatus)
	}
}

func truncateBody(data []byte) []byte {
	if len(data) <= maxErrorBody {
		return data
	}
	return data[:maxErrorBody]
}
