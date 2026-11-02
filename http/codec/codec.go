package codec

import "encoding/json"

// Codec 定义请求体编码与响应体解码的契约。
type Codec interface {
	ContentType() string
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}

// JSON 是默认的 JSON 编解码实现。
type JSON struct{}

var _ Codec = JSON{}

func (JSON) ContentType() string { return "application/json" }

func (JSON) Marshal(v any) ([]byte, error) { return json.Marshal(v) }

func (JSON) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }

// Default 是 client 默认使用的编解码器。
var Default Codec = JSON{}
