package base

// Error 是统一错误契约：内嵌官方 error，并额外要求 Code()。
type Error interface {
	error
	Code() Code
}

// CustomError 是 Error 契约的默认实现，携带错误码与错误信息。
// 预定义错误以本类型字面量声明；业务可直接填充 ErrCode 与 Msg 自行实现。
type CustomError struct {
	ErrCode Code
	Msg     string
}

var (
	_ error = CustomError{}
	_ Error = CustomError{}

	// Custom 是预留的自定义错误占位，错误码为 CodeUnset(-1)，留给后续按业务填充。
	Custom = CustomError{ErrCode: CodeUnset, Msg: ""}
)

// Error 返回错误信息，满足官方 error 接口。
func (e CustomError) Error() string { return e.Msg }

// Code 返回错误码，满足 Error 契约。
func (e CustomError) Code() Code { return e.ErrCode }

// WithCustomMsg 返回替换了错误信息、错误码保持不变的副本。
func (e CustomError) WithCustomMsg(msg string) CustomError {
	e.Msg = msg
	return e
}
