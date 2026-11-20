// Package backend 定义 TokenHub 服务端的统一业务错误枚举。
//
// 错误按“归属层级”划为 4 个模块,每个模块一个文件:
//
//	Common   通用/框架层  common.go
//	Auth     准入/安全层  auth.go
//	Business 业务规则层  business.go
//	Runtime  运行/系统层  runtime.go
//
// 错误码 = 模块号 + 3 位序号,由 base.NewCode 生成。
// 新增错误必须先在下方判据里确定层级,再写入对应文件,禁止跨层混放。
package backend

import "github.com/hktkhub/commonsdk/error/base"

const (
	// ModuleCommon 通用/框架层,编号 1。
	// 收录:与业务域无关、对所有模块通用的错误。
	// 例:参数校验、请求格式、序列化、分页/排序参数、通用未找到/冲突、内部未知错误兜底。
	// 判据:错误描述中不出现任何业务域名词。
	ModuleCommon base.Module = 1

	// ModuleAuth 准入/安全层,编号 2。
	// 收录:决定请求能否被允许进入或执行的认证与授权错误。
	// 例:401 认证(令牌/会话/外部登录)、403 授权(无权/用户禁用/scope 不足)、安全约束冲突、风控审计拦截。
	// 判据:语义是“调用方没有资格”,而非业务状态不对或依赖故障。
	ModuleAuth base.Module = 2

	// ModuleBusiness 业务规则层,编号 3。
	// 收录:业务实体与业务规则校验失败,且用户可理解、理论上可纠正的前置条件。
	// 例:租户/权益/分类、资金(积分/定价/预冻结)、额度(会员/APIKey/租户/素材)、对账/开票、分销/合作方。
	// 判据:错误由业务规则触发,不涉及认证、网络或依赖。
	ModuleBusiness base.Module = 3

	// ModuleRuntime 运行/系统层,编号 4。
	// 收录:非业务、用户不可纠正的系统侧或外部依赖问题,前 3 层的兜底。
	// 例:网关转发、模型不可用、配置缺失、限流、缓存/原子状态不可用、下游调用失败、基础设施不可用。
	// 判据:错误来自系统或外部依赖,不是用户操作能改变的。
	ModuleRuntime base.Module = 4
)
