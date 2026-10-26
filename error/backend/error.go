package backend

import "github.com/hktkhub/commonsdk/error/base"

// backendErr 是 backend 各模块错误枚举的统一类型。
// 作为非导出类型，它只通过 base.Error 契约对外流动，外部按 error / base.Error 使用。
type backendErr struct {
	ErrCode base.Code
	Msg     string
}

var (
	_ error      = backendErr{}
	_ base.Error = backendErr{}
)

// Error 返回错误信息，满足官方 error 接口。
func (e backendErr) Error() string { return e.Msg }

// Code 返回错误码，满足 base.Error 契约。
func (e backendErr) Code() base.Code { return e.ErrCode }

// WithCustomMsg 返回替换了错误信息、错误码保持不变的副本。
func (e backendErr) WithCustomMsg(msg string) backendErr {
	e.Msg = msg
	return e
}
