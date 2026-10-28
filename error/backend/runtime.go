// runtime.go 归 ModuleRuntime(运行/系统层)。
// 新增判据:来自系统或外部依赖、用户不可纠正的问题,前 3 层都排除后的兜底。
// 典型:网关转发、模型不可用、配置缺失、限流、缓存/原子状态不可用、下游调用失败、审核下游未配置。
// 反例:能被用户操作纠正的,退回 business.go。
package backend

import "github.com/hktkhub/commonsdk/error/base"

var (
	// 真实来源:domain.ErrRuntimeCacheUnavailable。
	RuntimeCacheUnavailable = backendErr{ErrCode: base.NewCode(ModuleRuntime, 1), Msg: "运行时缓存依赖暂不可用"}
	// 真实来源:domain.ErrRuntimeCacheCorrupt。
	RuntimeCacheCorrupt = backendErr{ErrCode: base.NewCode(ModuleRuntime, 2), Msg: "运行时缓存数据损坏"}
	// 真实来源:domain.ErrAtomicStateUnavailable(fail-closed,不得降级放行)。
	RuntimeAtomicStateUnavailable = backendErr{ErrCode: base.NewCode(ModuleRuntime, 3), Msg: "原子状态运行时不可用"}
	// 真实来源:api/respond/errors.go ErrorCodeTokenHubRuntimeConfigMissing=10509。
	RuntimeConfigMissing = backendErr{ErrCode: base.NewCode(ModuleRuntime, 4), Msg: "运行时配置缺失"}
	// 真实来源:api/respond/errors.go ErrorCodeTokenHubRuntimeModelUnavailable=10510。
	RuntimeModelUnavailable = backendErr{ErrCode: base.NewCode(ModuleRuntime, 5), Msg: "运行时模型不可用"}
	// 真实来源:api/respond/errors.go ErrorCodeTokenHubRuntimeInternalError=10511。
	RuntimeInternalError = backendErr{ErrCode: base.NewCode(ModuleRuntime, 6), Msg: "运行时内部错误"}
	// 真实来源:api/respond/errors.go ErrorCodeDownstreamPeanut=10508。
	RuntimeDownstreamPeanut = backendErr{ErrCode: base.NewCode(ModuleRuntime, 7), Msg: "Peanut 下游运行时调用失败"}
	// 真实来源:api/respond/errors.go ErrorCodeRuntimeInfraUnavailable=10519。
	RuntimeInfraUnavailable = backendErr{ErrCode: base.NewCode(ModuleRuntime, 8), Msg: "运行时基础设施不可用"}
	// 真实来源:api/respond/errors.go ErrorCodeRateLimited=10520。
	RuntimeRateLimited = backendErr{ErrCode: base.NewCode(ModuleRuntime, 9), Msg: "请求被限流"}
	// 真实来源:api/respond/errors.go ErrorCodeClientClosedRequest=10513。
	RuntimeClientClosedRequest = backendErr{ErrCode: base.NewCode(ModuleRuntime, 10), Msg: "客户端提前断开连接"}
	// 真实来源:domain.ErrModerationBatchSizeExceeded。
	ModerationBatchSizeExceeded = backendErr{ErrCode: base.NewCode(ModuleRuntime, 11), Msg: "审核批量创建超出数量上限"}
	// 真实来源:domain.ErrModerationClientNotConfigured。
	ModerationClientNotConfigured = backendErr{ErrCode: base.NewCode(ModuleRuntime, 12), Msg: "审核下游客户端未配置"}
	// 真实来源:domain.ErrModerationResultNotFound。
	ModerationResultNotFoundUpstream = backendErr{ErrCode: base.NewCode(ModuleRuntime, 13), Msg: "上游未找到审核结果"}
	// 真实来源:domain.ErrGetModerationNotEnabled。
	ModerationResultsNotEnabled = backendErr{ErrCode: base.NewCode(ModuleRuntime, 14), Msg: "该租户未开通审核结果查询"}
	// 真实来源:domain.ErrModerationResultOutdated。
	ModerationResultOutdated = backendErr{ErrCode: base.NewCode(ModuleRuntime, 15), Msg: "审核结果已超出可查询窗口"}
)
