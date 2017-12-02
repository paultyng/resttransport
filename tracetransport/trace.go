package tracetransport

import (
	"context"

	"cloud.google.com/go/trace"
	"github.com/pkg/errors"

	"github.com/paultyng/resttransport"
	"github.com/paultyng/resttransport/routename"
)

type tracingTransport struct {
	inner  resttransport.Transport
	namer  routename.Namer
	tracer *trace.Client
}

// New returns a new instance of a resttransport that implements opentracing.
func New(tracer *trace.Client, inner resttransport.Transport) resttransport.Transport {
	return &tracingTransport{
		inner:  inner,
		namer:  routename.New(),
		tracer: tracer,
	}
}

func (t *tracingTransport) RegisterHandler(httpMethod, path string, consumes []string, h resttransport.Handler) error {
	return t.inner.RegisterHandler(httpMethod, path, consumes, t.wrapHandler(httpMethod, path, h))
}

func (t *tracingTransport) RegisterAuthenticatedHandler(httpMethod, path string, consumes []string, h resttransport.Handler) error {
	return t.inner.RegisterAuthenticatedHandler(httpMethod, path, consumes, t.wrapHandler(httpMethod, path, h))
}

// https://github.com/GoogleCloudPlatform/google-cloud-go/blob/32a444f1bdd6d9313e6c82d90e66b599a2caa285/trace/trace.go#L171
const (
	httpHeader = `X-Cloud-Trace-Context`
)

func (t *tracingTransport) wrapHandler(httpMethod, path string, inner resttransport.Handler) resttransport.Handler {
	return func(ctx context.Context, reqres resttransport.RequestResponse) error {
		spanName := t.namer.Name(httpMethod, path)
		span := t.tracer.SpanFromHeader(spanName, reqres.RequestHeader().Get(httpHeader))
		defer span.Finish()

		span.SetLabel(trace.LabelHTTPMethod, httpMethod)
		span.SetLabel(trace.LabelHTTPURL, path)

		ctx = trace.NewContext(ctx, span)

		err := inner(ctx, reqres)
		if err != nil {
			span.SetLabel("err", err.Error())
			span.SetLabel("cause", errors.Cause(err).Error())
		}

		return err
	}
}
