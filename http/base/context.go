package base

import "context"

type metadataKey struct{}

// WithMetadata 将元数据写入 context，供出站 transport 统一提取。
// Extra 会被复制一份，避免调用方后续修改 map 影响已在链路中的元数据。
func WithMetadata(ctx context.Context, md Metadata) context.Context {
	if len(md.Extra) > 0 {
		extra := make(map[string]string, len(md.Extra))
		for name, value := range md.Extra {
			extra[name] = value
		}
		md.Extra = extra
	}
	return context.WithValue(ctx, metadataKey{}, md)
}

// MetadataFromContext 读取 context 中的元数据。
func MetadataFromContext(ctx context.Context) (Metadata, bool) {
	if ctx == nil {
		return Metadata{}, false
	}
	md, ok := ctx.Value(metadataKey{}).(Metadata)
	if !ok || md.IsZero() {
		return Metadata{}, false
	}
	return md, true
}
