package tracetransport

import (
	"context"
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"

	"github.com/paultyng/resttransport"
	"github.com/paultyng/resttransport/routename"
)

type tracingTransport struct {
	inner  resttransport.Transport
	namer  routename.Namer
	tracer opentracing.Tracer
}

// New returns a new instance of a resttransport that implements opentracing.
func New(tracer opentracing.Tracer, inner resttransport.Transport) resttransport.Transport {
	return &tracingTransport{
		inner:  inner,
		namer:  routename.New(),
		tracer: tracer,
	}
}

func (t *tracingTransport) RegisterHandler(httpMethod, path string, h resttransport.Handler) error {
	return t.inner.RegisterHandler(httpMethod, path, t.wrapHandler(httpMethod, path, h))
}

func (t *tracingTransport) RegisterAuthenticatedHandler(httpMethod, path string, h resttransport.Handler) error {
	return t.inner.RegisterAuthenticatedHandler(httpMethod, path, t.wrapHandler(httpMethod, path, h))
}

func (t *tracingTransport) wrapHandler(httpMethod, path string, inner resttransport.Handler) resttransport.Handler {
	return func(ctx context.Context, reqres resttransport.RequestResponse) error {
		wireContext, _ := t.tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(reqres.RequestHeader()))

		spanName := t.namer.Name(httpMethod, path)
		span := t.tracer.StartSpan(
			spanName,
			ext.RPCServerOption(wireContext),
		)
		defer span.Finish()

		ext.HTTPUrl.Set(span, path)
		ext.HTTPMethod.Set(span, httpMethod)

		ctx = opentracing.ContextWithSpan(ctx, span)

		err := inner(ctx, reqres)
		if err != nil {
			span.LogFields(
				log.String("err", err.Error()),
				log.String("cause", errors.Cause(err).Error()),
			)
		}

		return err
	}
}

// InjectHTTP propagates tracing headers to child HTTP requests.
func InjectHTTP(ctx context.Context, outbound *http.Request) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		span.Tracer().Inject(span.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(outbound.Header))
	}
}

//TODO: InjectGRPC, see https://github.com/go-kit/kit/blob/master/tracing/opentracing/grpc.go
