package base

import "fmt"

const (
	moduleDigits  = 2
	errcodeDigits = 3
	prefixDigits  = 7

	localWidth = moduleDigits + errcodeDigits // 本地码 5 位
	fullWidth  = prefixDigits + localWidth    // 完整码 12 位

	maxModule  = 99
	maxErrcode = 999

	moduleScale = 1000
	prefixScale = 100000
)

// CodeUnset 表示未指定错误码。业务自定义错误在未显式指定 ErrCode 时使用。
const CodeUnset Code = -1

// Module 是本地错误码中的模块号，占 2 位。
type Module int64

// Code 是本地错误码，由 2 位模块号 + 3 位错误码组成。
// 采用有符号类型，以便用 -1（CodeUnset）表示"未指定"。
type Code int64

// Prefix 是业务线 + 服务的 7 位前缀，当前仅预留接口，暂不注入。
type Prefix int64

// FullCode 是 7 位前缀 + 5 位本地码组合出的完整错误码，占 12 位。
type FullCode int64

// NewCode 组合模块号与错误码，生成本地错误码。
// module 越界 [0, 99] 或 errcode 越界 [0, 999] 时 panic，避免污染码位。
func NewCode(module Module, errcode uint16) Code {
	if module < 0 || module > maxModule {
		panic(fmt.Sprintf("base: module %d out of range [0, %d]", module, maxModule))
	}
	if errcode > maxErrcode {
		panic(fmt.Sprintf("base: errcode %d out of range [0, %d]", errcode, maxErrcode))
	}
	return Code(module*moduleScale + Module(errcode))
}

// Module 返回错误码所属的模块号。
func (c Code) Module() Module { return Module(c / moduleScale) }

// Errcode 返回模块内的错误码。
func (c Code) Errcode() uint16 { return uint16(c % moduleScale) }

// String 返回定长 5 位的本地错误码字符串。
func (c Code) String() string { return fmt.Sprintf("%0*d", localWidth, c) }

// Combine 将 7 位前缀与本地错误码组合为完整错误码。
// 当前仅保留接口，不做实际前缀注入。
func (p Prefix) Combine(c Code) FullCode { return FullCode(p)*prefixScale + FullCode(c) }

// String 返回定长 12 位的完整错误码字符串。
func (f FullCode) String() string { return fmt.Sprintf("%0*d", fullWidth, f) }
