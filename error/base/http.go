package base

// 模块 00：标准 HTTP 语义，永久保留，错误码位直接承载 HTTP 状态码（如 404 -> 00404）。
// 不逐一枚举状态值，只保留一个构造入口，状态码按需传入。

const ModuleHTTP Module = 0 // HTTP/标准语义。

// HTTPError 是模块 00 专用的错误类型，错误码位承载 HTTP 状态码，如 404 -> 00404。
// 模块 0 永久保留给 HTTP，业务模块不得占用。
type HTTPError struct {
	Status int
	Msg    string
}

var (
	_ error = HTTPError{}
	_ Error = HTTPError{}
)

// Error 返回错误信息，满足官方 error 接口。
func (e HTTPError) Error() string { return e.Msg }

// Code 返回本地错误码，HTTP 状态码即模块内的错误码。
func (e HTTPError) Code() Code { return NewCode(ModuleHTTP, uint16(e.Status)) }

// 模块 00：标准 HTTP 语义下的常用错误。
var (
	HTTPBadRequest            = HTTPError{Status: 400, Msg: "请求参数错误"}
	HTTPUnauthorized          = HTTPError{Status: 401, Msg: "未认证或凭证无效"}
	HTTPForbidden             = HTTPError{Status: 403, Msg: "禁止访问"}
	HTTPNotFound              = HTTPError{Status: 404, Msg: "资源不存在"}
	HTTPMethodNotAllowed      = HTTPError{Status: 405, Msg: "请求方法不被允许"}
	HTTPConflict              = HTTPError{Status: 409, Msg: "资源冲突"}
	HTTPGone                  = HTTPError{Status: 410, Msg: "资源已失效"}
	HTTPRequestEntityTooLarge = HTTPError{Status: 413, Msg: "请求体过大"}
	HTTPUnprocessableEntity   = HTTPError{Status: 422, Msg: "请求无法处理"}
	HTTPTooManyRequests       = HTTPError{Status: 429, Msg: "请求过于频繁"}
	HTTPInternalServerError   = HTTPError{Status: 500, Msg: "服务器内部错误"}
	HTTPNotImplemented        = HTTPError{Status: 501, Msg: "功能未实现"}
	HTTPBadGateway            = HTTPError{Status: 502, Msg: "网关错误"}
	HTTPServiceUnavailable    = HTTPError{Status: 503, Msg: "服务暂不可用"}
	HTTPGatewayTimeout        = HTTPError{Status: 504, Msg: "网关超时"}
)
