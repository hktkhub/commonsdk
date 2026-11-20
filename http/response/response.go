package response

import "reflect"

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

// FromError 把实现了 Code() 的错误转换为错误响应。
// Code 支持任意整数底层类型（包括 error/base.Code），Msg 取 Error()。
// 普通 error 没有 Code() 时业务码为 0，可使用 FromErrorWithCode 指定兜底码。
func FromError(err error) Envelope[any] {
	return FromErrorWithCode(0, err)
}

// FromErrorWithCode 与 FromError 类似，但错误没有 Code() 时使用 fallback 作为业务码。
func FromErrorWithCode(fallback int, err error) Envelope[any] {
	if err == nil {
		return OK(nil)
	}
	code, ok := errorCode(err)
	if !ok {
		code = fallback
	}
	return Envelope[any]{Code: code, Msg: err.Error()}
}

func errorCode(err error) (int, bool) {
	if err == nil {
		return 0, false
	}
	value := reflect.ValueOf(err)
	if !value.IsValid() || (value.Kind() == reflect.Ptr && value.IsNil()) {
		return 0, false
	}
	method := value.MethodByName("Code")
	if !method.IsValid() && value.Kind() != reflect.Ptr {
		ptr := reflect.New(value.Type())
		ptr.Elem().Set(value)
		method = ptr.MethodByName("Code")
	}
	if !method.IsValid() || method.Type().NumIn() != 0 || method.Type().NumOut() != 1 {
		return 0, false
	}
	result := method.Call(nil)[0]
	switch result.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(result.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(result.Uint()), true
	default:
		return 0, false
	}
}
