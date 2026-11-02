package response

// Envelope 是服务端与客户端的统一响应契约。
type Envelope[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

// OK 构造成功响应。
func OK(data any) Envelope[any] {
	return Envelope[any]{Code: 0, Msg: "ok", Data: data}
}

// Error 构造错误响应。
func Error(code int, msg string) Envelope[any] {
	return Envelope[any]{Code: code, Msg: msg}
}
