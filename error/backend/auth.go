// auth.go 归 ModuleAuth(准入/安全层)。
// 新增判据:错误决定“请求能否被允许”,即认证(401)或授权(403)语义。
// 典型:令牌/会话/外部登录、无权访问、用户禁用、scope 不足、安全约束冲突、风控/审计拦截。
// 反例:业务状态冲突放 business.go;依赖故障放 runtime.go。
package backend

import "github.com/hktkhub/commonsdk/error/base"

var (
	// 真实来源:domain.ErrUnauthorized。
	AuthUnauthorized = backendErr{ErrCode: base.NewCode(ModuleAuth, 1), Msg: "未认证"}
	// 真实来源:domain.ErrTokenExpired。
	AuthTokenExpired = backendErr{ErrCode: base.NewCode(ModuleAuth, 2), Msg: "令牌已过期"}
	// 真实来源:domain.ErrTokenRevoked。
	AuthTokenRevoked = backendErr{ErrCode: base.NewCode(ModuleAuth, 3), Msg: "令牌已被吊销"}
	// 真实来源:domain.ErrForbidden。
	AuthForbidden = backendErr{ErrCode: base.NewCode(ModuleAuth, 4), Msg: "无权访问该资源"}
	// 真实来源:domain.ErrUserDisabled。
	AuthUserDisabled = backendErr{ErrCode: base.NewCode(ModuleAuth, 5), Msg: "用户已被禁用"}
	// 真实来源:domain.ErrPhoneVerificationInvalid。
	AuthPhoneVerificationInvalid = backendErr{ErrCode: base.NewCode(ModuleAuth, 6), Msg: "手机验证码无效、过期或已使用"}
	// 真实来源:domain.ErrPhoneLoginUnavailable。
	AuthPhoneLoginUnavailable = backendErr{ErrCode: base.NewCode(ModuleAuth, 7), Msg: "该手机号当前不可用于登录"}
	// 真实来源:domain.ErrPhoneRegistrationUnavailable。
	AuthPhoneRegistrationUnavailable = backendErr{ErrCode: base.NewCode(ModuleAuth, 8), Msg: "该手机号当前不可用于注册"}
	// 真实来源:domain.ErrPhoneAlreadyRegistered。
	AuthPhoneAlreadyRegistered = backendErr{ErrCode: base.NewCode(ModuleAuth, 9), Msg: "该手机号已注册"}
	// 真实来源:domain.ErrPhoneInCoolingPeriod。
	AuthPhoneInCoolingPeriod = backendErr{ErrCode: base.NewCode(ModuleAuth, 10), Msg: "该手机号处于账户注销冷却期"}
	// 真实来源:domain.ErrEmailVerificationInvalid。
	AuthEmailVerificationInvalid = backendErr{ErrCode: base.NewCode(ModuleAuth, 11), Msg: "邮箱验证码无效、过期或已使用"}
	// 真实来源:domain.ErrEmailLoginUnavailable。
	AuthEmailLoginUnavailable = backendErr{ErrCode: base.NewCode(ModuleAuth, 12), Msg: "该邮箱当前不可用于登录"}
	// 真实来源:domain.ErrEmailRegistrationUnavailable。
	AuthEmailRegistrationUnavailable = backendErr{ErrCode: base.NewCode(ModuleAuth, 13), Msg: "该邮箱当前不可用于注册"}
	// 真实来源:domain.ErrEmailAlreadyRegistered。
	AuthEmailAlreadyRegistered = backendErr{ErrCode: base.NewCode(ModuleAuth, 14), Msg: "该邮箱已注册"}
	// 真实来源:domain.ErrEmailInCoolingPeriod。
	AuthEmailInCoolingPeriod = backendErr{ErrCode: base.NewCode(ModuleAuth, 15), Msg: "该邮箱处于账户注销冷却期"}
	// 真实来源:domain.ErrVerificationResendNotReady。
	AuthVerificationResendNotReady = backendErr{ErrCode: base.NewCode(ModuleAuth, 16), Msg: "验证码重发冷却中"}
	// 真实来源:domain.ErrExternalLoginStateInvalid。
	AuthExternalLoginStateInvalid = backendErr{ErrCode: base.NewCode(ModuleAuth, 17), Msg: "外部登录回调状态无效或已过期"}
	// 真实来源:domain.ErrExternalProviderUnavailable。
	AuthExternalProviderUnavailable = backendErr{ErrCode: base.NewCode(ModuleAuth, 18), Msg: "外部登录提供方暂不可用"}
	// 真实来源:domain.ErrExternalAccountDisabled。
	AuthExternalAccountDisabled = backendErr{ErrCode: base.NewCode(ModuleAuth, 19), Msg: "绑定的外部账户已被禁用"}
	// 真实来源:domain.ErrExternalAccountAlreadyLinked。
	AuthExternalAccountAlreadyLinked = backendErr{ErrCode: base.NewCode(ModuleAuth, 20), Msg: "该外部账户已绑定其他平台用户"}
	// 真实来源:api/respond/errors.go ErrorCodeInsufficientScope=10406。
	AuthInsufficientScope = backendErr{ErrCode: base.NewCode(ModuleAuth, 21), Msg: "缺少所需授权范围"}
	// 真实来源:api/respond/errors.go ErrorCodeSecurityConstraint=10501(由 domain.ErrFailedPrecondition 映射)。
	AuthConstraintViolation = backendErr{ErrCode: base.NewCode(ModuleAuth, 22), Msg: "安全约束校验未通过"}
)
