// Package client 是团队出站 HTTP 请求的唯一入口。
//
// 业务只通过本包发起请求，并把链路元数据放在 context 中：
//
//	ctx := base.WithMetadata(ctx, base.Metadata{
//		TraceID: "4bf92f3577b34da6a3ce929d0e0e4736",
//		SpanID:  "00f067aa0ba902b7",
//		Lane:    "lane-a",
//	})
//	result, err := c.DoJSON(ctx, http.MethodPost, "/api/v1/demo", in, &out)
//
// trace/泳道 header 由 transport 自动注入，业务无需手工设置。
package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hktkhub/commonsdk/http/codec"
	"github.com/hktkhub/commonsdk/http/transport"
)

const maxErrorBody = 8 << 10

// Client 是官方 http.Client 的受控包装。
type Client struct {
	hc          *http.Client
	baseURL     *url.URL
	codec       codec.Codec
	successCode int
	envelope    bool
}

// New 构造 Client，并确保底层请求经过统一注入逻辑。
func New(opts ...Option) (*Client, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	var baseURL *url.URL
	if cfg.baseURL != "" {
		u, err := url.Parse(cfg.baseURL)
		if err != nil {
			return nil, fmt.Errorf("http/client: invalid base url: %w", err)
		}
		if u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf("http/client: base url must be absolute: %q", cfg.baseURL)
		}
		baseURL = u
	}

	if cfg.codec == nil {
		cfg.codec = codec.Default
	}

	rt := transport.New(
		cfg.transport,
		transport.WithUserAgent(cfg.userAgent),
		transport.WithWhitelist(cfg.propagation...),
		transport.WithHeaderNames(cfg.headerNames),
	)

	return &Client{
		hc:          &http.Client{Timeout: cfg.timeout, Transport: rt},
		baseURL:     baseURL,
		codec:       cfg.codec,
		successCode: cfg.successCode,
		envelope:    cfg.envelope,
	}, nil
}

// NewRequest 构造请求并解析相对路径。
func (c *Client) NewRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	u, err := c.resolve(path)
	if err != nil {
		return nil, err
	}
	return http.NewRequestWithContext(ctx, method, u.String(), body)
}

// Do 发送请求。相对 URL 会基于 BaseURL 解析。
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("http/client: nil request")
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	if c.baseURL != nil && !req.URL.IsAbs() {
		u, err := c.resolve(req.URL.String())
		if err != nil {
			return nil, err
		}
		req = req.Clone(req.Context())
		req.URL = u
	}
	return c.hc.Do(req)
}

// Get 发送 GET 请求。
func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	req, err := c.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

// Post 发送 POST 请求。
func (c *Client) Post(ctx context.Context, path, contentType string, body io.Reader) (*http.Response, error) {
	req, err := c.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.Do(ctx, req)
}

// DoJSON 以 JSON 编解码请求体与响应体，并按统一响应 {code,msg,data} 解析。
// 无论业务是否成功，只要拿到了 HTTP 响应就会返回 Result；
// 业务码非成功或 HTTP 状态 >= 400 时同时返回 *APIError。
func (c *Client) DoJSON(ctx context.Context, method, path string, in, out any) (*Result, error) {
	var body io.Reader
	if in != nil {
		data, err := c.codec.Marshal(in)
		if err != nil {
			return nil, fmt.Errorf("http/client: encode request: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := c.NewRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	if in != nil {
		req.Header.Set("Content-Type", c.codec.ContentType())
	}
	req.Header.Set("Accept", c.codec.ContentType())

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := &Result{
		HTTPStatus: resp.StatusCode,
		Header:     resp.Header.Clone(),
		codec:      c.codec,
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("http/client: read response: %w", err)
	}
	trimmed := bytes.TrimSpace(data)
	hasBody := len(trimmed) > 0

	decodedEnvelope := false
	if hasBody && c.envelope {
		var env envelope
		if err := c.codec.Unmarshal(trimmed, &env); err != nil {
			if resp.StatusCode >= http.StatusBadRequest {
				return result, &APIError{
					Msg:        resp.Status,
					HTTPStatus: resp.StatusCode,
					Body:       truncateBody(data),
				}
			}
			return result, fmt.Errorf("http/client: decode envelope: %w", err)
		}
		result.Code = env.Code
		result.Msg = env.Msg
		result.Data = env.Data
		decodedEnvelope = true
	} else if hasBody {
		result.Data = data
	}

	if resp.StatusCode >= http.StatusBadRequest {
		msg := result.Msg
		if msg == "" {
			msg = resp.Status
		}
		return result, &APIError{
			Code:       result.Code,
			Msg:        msg,
			HTTPStatus: resp.StatusCode,
			Body:       truncateBody(data),
		}
	}

	if decodedEnvelope && result.Code != c.successCode {
		return result, &APIError{
			Code:       result.Code,
			Msg:        result.Msg,
			HTTPStatus: resp.StatusCode,
		}
	}

	if out != nil {
		if err := result.Decode(out); err != nil {
			return result, fmt.Errorf("http/client: decode response: %w", err)
		}
	}
	return result, nil
}

func (c *Client) resolve(path string) (*url.URL, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("http/client: invalid url %q: %w", path, err)
	}
	if u.IsAbs() || c.baseURL == nil {
		return u, nil
	}
	return c.baseURL.ResolveReference(u), nil
}
