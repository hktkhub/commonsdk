// common.go 归 ModuleCommon(通用/框架层)。
// 新增判据:错误与业务域无关、对所有模块通用,描述里不出现业务名词。
// 典型:参数校验、请求格式、序列化、分页/排序参数、通用未找到/冲突、内部未知错误兜底。
// 反例:涉及租户/资金/令牌等业务名词的,一律不放这里。
package backend

import "github.com/hktkhub/commonsdk/error/base"

var (
	// 真实来源:domain.ErrNotFound。
	CommonNotFound = backendErr{ErrCode: base.NewCode(ModuleCommon, 1), Msg: "资源不存在"}
	// 真实来源:domain.ErrInvalidInput。
	CommonInvalidInput = backendErr{ErrCode: base.NewCode(ModuleCommon, 2), Msg: "请求参数非法"}
	// 真实来源:domain.ErrConflict(包裹 ErrFailedPrecondition)。
	CommonConflict = backendErr{ErrCode: base.NewCode(ModuleCommon, 3), Msg: "写入与现有状态冲突"}
	// 真实来源:domain.ErrFailedPrecondition。
	CommonFailedPrecondition = backendErr{ErrCode: base.NewCode(ModuleCommon, 4), Msg: "缺少必要的前置条件"}
	// 真实来源:domain.ErrInteractionExpired。
	CommonInteractionExpired = backendErr{ErrCode: base.NewCode(ModuleCommon, 5), Msg: "交互已过期,无法再提交"}
	// 真实来源:domain.ErrAlreadyCompleted。
	CommonAlreadyCompleted = backendErr{ErrCode: base.NewCode(ModuleCommon, 6), Msg: "操作目标已完成,当前操作无法执行"}
)
