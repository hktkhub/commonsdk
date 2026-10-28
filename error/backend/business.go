// business.go 归 ModuleBusiness(业务规则层)。
// 新增判据:由业务规则触发、用户可理解且理论上可纠正的前置条件。
// 分节:租户/权益、资金、额度、对账/开票/凭证、分销/合作方。
// 反例:认证授权放 auth.go;网络/依赖故障放 runtime.go。
package backend

import "github.com/hktkhub/commonsdk/error/base"

var (
	// ---- 租户/权益 ----
	// 真实来源:domain.ErrTenantNameConflict。
	TenantNameConflict = backendErr{ErrCode: base.NewCode(ModuleBusiness, 1), Msg: "租户名称已存在"}
	// 真实来源:domain.ErrTenantsAlreadyBound。
	TenantAlreadyBoundToAgreement = backendErr{ErrCode: base.NewCode(ModuleBusiness, 2), Msg: "租户已绑定生效期重叠的协议"}
	// 真实来源:domain.ErrAgreementNoAlreadyExists。
	AgreementNoAlreadyExists = backendErr{ErrCode: base.NewCode(ModuleBusiness, 3), Msg: "协议编号已存在"}
	// 真实来源:domain.ErrTenantTargetProviderRuleNotFound。
	TenantTargetProviderRuleNotFound = backendErr{ErrCode: base.NewCode(ModuleBusiness, 4), Msg: "租户目标供应商规则不存在"}
	// 真实来源:domain.ErrTenantTargetProviderRuleDuplicate。
	TenantTargetProviderRuleDuplicate = backendErr{ErrCode: base.NewCode(ModuleBusiness, 5), Msg: "该租户与供应商的目标规则已存在"}
	// 真实来源:domain.ErrTenantTargetProviderRuleManagedBySale。
	TenantTargetProviderRuleManagedBySale = backendErr{ErrCode: base.NewCode(ModuleBusiness, 6), Msg: "该目标规则由上游供应商销售托管,不可直接修改"}
	// 真实来源:domain.ErrUpstreamProviderSaleTenantNotBound。
	TenantNotBoundToUpstreamProviderSale = backendErr{ErrCode: base.NewCode(ModuleBusiness, 7), Msg: "租户未绑定该上游供应商销售"}
	// 真实来源:domain.AmbiguousPermissionKeyError。
	PermissionKeyAmbiguous = backendErr{ErrCode: base.NewCode(ModuleBusiness, 8), Msg: "权限键跨多个域存在歧义"}
	// 真实来源:domain.ErrEntitlementNotFound。
	EntitlementNotFound = backendErr{ErrCode: base.NewCode(ModuleBusiness, 9), Msg: "未解析到可用权益"}
	// 真实来源:domain.ErrCategoryNotFound。
	EntitlementCategoryNotFound = backendErr{ErrCode: base.NewCode(ModuleBusiness, 10), Msg: "权益分类不存在"}
	// 真实来源:domain.ErrExclusiveStrategyNotFound。
	EntitlementExclusiveStrategyNotFound = backendErr{ErrCode: base.NewCode(ModuleBusiness, 11), Msg: "专属策略权益不存在"}
	// 真实来源:domain.ErrModelNotAccessibleByEntitlement。
	ModelNotAccessibleByEntitlement = backendErr{ErrCode: base.NewCode(ModuleBusiness, 12), Msg: "当前权益不允许访问该模型"}

	// ---- 资金 ----
	// 真实来源:domain.ErrInsufficientCredits。
	BillingInsufficientCredits = backendErr{ErrCode: base.NewCode(ModuleBusiness, 13), Msg: "账户积分余额不足"}
	// 真实来源:domain.ErrPricingRuleNotFound。
	BillingPricingRuleNotFound = backendErr{ErrCode: base.NewCode(ModuleBusiness, 14), Msg: "未找到适用的定价规则"}
	// 真实来源:domain.ErrBillingPreholdFailed。
	BillingPreholdFailed = backendErr{ErrCode: base.NewCode(ModuleBusiness, 15), Msg: "运行时计费预冻结失败"}
	// 真实来源:api/respond/errors.go ErrorCodePlatformInsufficientCredits=10514。
	BillingPlatformInsufficientCredits = backendErr{ErrCode: base.NewCode(ModuleBusiness, 16), Msg: "平台上游账户余额不足"}
	// 真实来源:domain.ErrInvoiceInfoNotConfigured。
	BillingInvoiceInfoNotConfigured = backendErr{ErrCode: base.NewCode(ModuleBusiness, 17), Msg: "发起开票前需先配置生效中的企业开票信息"}

	// ---- 额度 ----
	// 真实来源:domain.ErrMembershipQuotaExceeded。
	QuotaMembershipExceeded = backendErr{ErrCode: base.NewCode(ModuleBusiness, 18), Msg: "已超出会员积分额度"}
	// 真实来源:domain.ErrMembershipInsufficientCredits。
	QuotaMembershipInsufficientCredits = backendErr{ErrCode: base.NewCode(ModuleBusiness, 19), Msg: "会员一次性积分余额已耗尽"}
	// 真实来源:domain.ErrMembershipPeriodicQuotaExceeded。
	QuotaMembershipPeriodicExceeded = backendErr{ErrCode: base.NewCode(ModuleBusiness, 20), Msg: "会员本周期额度已耗尽"}
	// 真实来源:domain.ErrAPIKeyQuotaExceeded。
	QuotaAPIKeyExceeded = backendErr{ErrCode: base.NewCode(ModuleBusiness, 21), Msg: "已超出 API Key 积分额度"}
	// 真实来源:domain.ErrAPIKeyInsufficientCredits。
	QuotaAPIKeyInsufficientCredits = backendErr{ErrCode: base.NewCode(ModuleBusiness, 22), Msg: "API Key 一次性积分余额已耗尽"}
	// 真实来源:domain.ErrAPIKeyPeriodicQuotaExceeded。
	QuotaAPIKeyPeriodicExceeded = backendErr{ErrCode: base.NewCode(ModuleBusiness, 23), Msg: "API Key 本周期额度已耗尽"}
	// 真实来源:domain.ErrTenantInsufficientCredits。
	QuotaTenantInsufficientCredits = backendErr{ErrCode: base.NewCode(ModuleBusiness, 24), Msg: "租户可用积分不足"}
	// 真实来源:domain.ErrSelfTenantQuotaExceeded。
	QuotaSelfTenantExceeded = backendErr{ErrCode: base.NewCode(ModuleBusiness, 25), Msg: "自注册租户数量已达上限"}
	// 真实来源:domain.ErrSeedanceAssetGroupLimitExceeded。
	QuotaSeedanceAssetGroupLimitExceeded = backendErr{ErrCode: base.NewCode(ModuleBusiness, 26), Msg: "素材组数量已达上限"}
	// 真实来源:domain.ErrSeedanceAssetLimitExceeded。
	QuotaSeedanceAssetLimitExceeded = backendErr{ErrCode: base.NewCode(ModuleBusiness, 27), Msg: "素材数量已达上限"}

	// ---- 对账/开票/凭证 ----
	// 真实来源:domain.ErrReconciliationBillAlreadyExists。
	ReconciliationBillAlreadyExists = backendErr{ErrCode: base.NewCode(ModuleBusiness, 28), Msg: "对账单已存在"}
	// 真实来源:domain.ErrPartnerReconciliationBillAlreadyExists。
	PartnerReconciliationBillAlreadyExists = backendErr{ErrCode: base.NewCode(ModuleBusiness, 29), Msg: "合作方对账单已存在"}
	// 真实来源:domain.ErrReconciliationBillStatusConflict。
	ReconciliationBillStatusConflict = backendErr{ErrCode: base.NewCode(ModuleBusiness, 30), Msg: "对账单状态冲突"}
	// 真实来源:domain.ErrReconciliationConfigNotInitialized。
	ReconciliationConfigNotInitialized = backendErr{ErrCode: base.NewCode(ModuleBusiness, 31), Msg: "对账配置未初始化"}
	// 真实来源:domain.ErrVoucherExportEmpty。
	VoucherExportEmpty = backendErr{ErrCode: base.NewCode(ModuleBusiness, 32), Msg: "该计费周期无凭证消费数据"}

	// ---- 分销/合作方 ----
	// 真实来源:domain.ErrPartnerEmailExists。
	PartnerEmailExists = backendErr{ErrCode: base.NewCode(ModuleBusiness, 33), Msg: "合作方邮箱已存在"}
	// 真实来源:domain.ErrPartnerNameExists。
	PartnerNameExists = backendErr{ErrCode: base.NewCode(ModuleBusiness, 34), Msg: "合作方名称已存在"}
	// 真实来源:domain.ErrPartnerNotFound。
	PartnerNotFound = backendErr{ErrCode: base.NewCode(ModuleBusiness, 35), Msg: "合作方不存在"}
	// 真实来源:domain.ErrPartnerInvalidStatus。
	PartnerInvalidStatus = backendErr{ErrCode: base.NewCode(ModuleBusiness, 36), Msg: "合作方状态非法"}
	// 真实来源:domain.ErrPartnerSessionNotFound。
	PartnerSessionNotFound = backendErr{ErrCode: base.NewCode(ModuleBusiness, 37), Msg: "合作方会话不存在"}
	// 真实来源:domain.ErrPartnerDisabled。
	PartnerDisabled = backendErr{ErrCode: base.NewCode(ModuleBusiness, 38), Msg: "合作方已被禁用"}
	// 真实来源:domain.ErrUpstreamProviderSaleNotFound。
	UpstreamProviderSaleNotFound = backendErr{ErrCode: base.NewCode(ModuleBusiness, 39), Msg: "上游供应商销售不存在"}
	// 真实来源:domain.ErrUpstreamProviderSaleNameDuplicate。
	UpstreamProviderSaleNameDuplicate = backendErr{ErrCode: base.NewCode(ModuleBusiness, 40), Msg: "上游供应商销售名称重复"}
	// 真实来源:domain.ErrUpstreamProviderSaleTenantNotBound。
	UpstreamProviderSaleTenantNotBound = backendErr{ErrCode: base.NewCode(ModuleBusiness, 41), Msg: "上游供应商销售未绑定该租户"}
	// 真实来源:domain.ErrUpstreamProviderSaleBindingConflict。
	UpstreamProviderSaleBindingConflict = backendErr{ErrCode: base.NewCode(ModuleBusiness, 42), Msg: "上游供应商销售绑定冲突"}
	// 真实来源:domain.ErrAIAppCenterConfigNameDuplicate。
	AIAppCenterConfigNameDuplicate = backendErr{ErrCode: base.NewCode(ModuleBusiness, 43), Msg: "AI 应用中心配置名称已存在"}
)
