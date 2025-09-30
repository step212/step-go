package trace

import (
	"context"
	"fmt"
	"time"

	"ariga.io/sqlcomment"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport"
	ktgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	trangrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	kthttp "github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/grpc"
)

const (
	IstioTraceId = "x-b3-traceid" // istio trace id
	IstioSpanId  = "x-b3-spanid"  // istio span id
)

const (
	IstioPrefix = "x-" // istio header prefix
)

// TraceID returns a traceid valuer.
func TraceID() log.Valuer {
	return func(ctx context.Context) interface{} {
		tp, _ := transport.FromServerContext(ctx)
		traceId := ""
		if tp == nil {
			return ""
		}
		switch tpKind := tp.Kind(); tpKind {
		case transport.KindHTTP:
			tpHttp := tp.(*http.Transport)
			traceId = tpHttp.RequestHeader().Get(IstioTraceId)
		case transport.KindGRPC:
			tpGrpc := tp.(*trangrpc.Transport)
			traceId = tpGrpc.RequestHeader().Get(IstioTraceId)
		}
		return traceId
	}
}

// SpanID returns a spanid valuer.
func SpanID() log.Valuer {
	return func(ctx context.Context) interface{} {
		tp, _ := transport.FromServerContext(ctx)
		if tp == nil {
			return ""
		}
		spanId := ""
		switch tpKind := tp.Kind(); tpKind {
		case transport.KindHTTP:
			tpHttp := tp.(*http.Transport)
			spanId = tpHttp.RequestHeader().Get(IstioSpanId)
		case transport.KindGRPC:
			tpGrpc := tp.(*trangrpc.Transport)
			spanId = tpGrpc.RequestHeader().Get(IstioSpanId)
		}
		return spanId
	}
}

// trace middleware: 将traceId添加到返回头里
func Server() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				// 断言成HTTP的Transport可以拿到特殊信息
				if ht, ok := tr.(*http.Transport); ok {
					traceId := ht.RequestHeader().Get(IstioTraceId)
					tr.ReplyHeader().Set(IstioTraceId, traceId)
				}
			}
			return handler(ctx, req)
		}
	}
}

// meta middleware: 设置metadata 默认前缀
func MetaServer() middleware.Middleware {
	return metadata.Server(metadata.WithPropagatedPrefix(IstioPrefix))
}

// DialInsecure returns an insecure GRPC connection.
func NewGrpcCli(ctx context.Context, url string, opts ...ktgrpc.ClientOption) (*grpc.ClientConn, error) {
	return ktgrpc.DialInsecure(ctx,
		ktgrpc.WithEndpoint(url),
		ktgrpc.WithTimeout(100*time.Second),
		ktgrpc.WithMiddleware(
			recovery.Recovery(),
			logging.Client(log.DefaultLogger),
			metadata.Client(metadata.WithPropagatedPrefix(IstioPrefix)),
		),
	)
}

// NewClient returns an HTTP client.
func NewHttpCli(ctx context.Context, url string, opts ...kthttp.ClientOption) (*kthttp.Client, error) {
	return kthttp.NewClient(ctx,
		kthttp.WithEndpoint(url),
		kthttp.WithTimeout(100*time.Second),
		kthttp.WithMiddleware(
			recovery.Recovery(),
			logging.Client(log.DefaultLogger),
			metadata.Client(metadata.WithPropagatedPrefix(IstioPrefix)),
		),
	)
}

// sql 注解里添加trace_id
type TraceIDCommenter struct{}

func (t TraceIDCommenter) Tag(ctx context.Context) sqlcomment.Tags {
	fn := TraceID()
	traceId := fn(ctx)
	return sqlcomment.Tags{
		"trace_id": fmt.Sprintf("%+v", traceId),
	}
}
