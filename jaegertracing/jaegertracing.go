package jaegertracing

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/n-creativesystem/go-fwncs"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go/config"
)

const componetName string = "fwncs/v0"

type Config struct {
	Tracer        opentracing.Tracer
	ComponentName string
}

func New() (io.Closer, error) {
	defcfg := config.Configuration{
		ServiceName: "fwncs-tracer",
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:            true,
			BufferFlushInterval: 1 * time.Second,
		},
	}
	cfg, err := defcfg.FromEnv()
	if err != nil {
		return nil, err
	}
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		return nil, err
	}
	opentracing.SetGlobalTracer(tracer)
	return closer, nil
}

func Tracing(tracer opentracing.Tracer) fwncs.HandlerFunc {
	return TracingWithConfig(Config{
		Tracer:        tracer,
		ComponentName: componetName,
	})
}

func TracingWithConfig(config Config) fwncs.HandlerFunc {
	if config.Tracer == nil {
		panic("trace middleware requires opentracing tracer")
	}
	if config.ComponentName == "" {
		config.ComponentName = componetName
	}

	return func(c fwncs.Context) {
		req := c.Request()
		opname := req.Method + " " + c.Path()
		var sp opentracing.Span
		tr := config.Tracer
		if ctx, err := tr.Extract(opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(req.Header)); err != nil {
			sp = tr.StartSpan(opname)
		} else {
			sp = tr.StartSpan(opname, ext.RPCServerOption(ctx))
		}
		ext.HTTPMethod.Set(sp, req.Method)
		ext.HTTPUrl.Set(sp, req.URL.String())
		ext.Component.Set(sp, config.ComponentName)
		req = req.WithContext(opentracing.ContextWithSpan(req.Context(), sp))
		c.SetRequest(req)
		c.Next()
	}
}

func CreateChildSpan(ctx context.Context, name string) opentracing.Span {
	parentSpan := opentracing.SpanFromContext(ctx)
	if parentSpan == nil {
		parentSpan = opentracing.StartSpan(name)
	}
	sp := opentracing.StartSpan(name, opentracing.ChildOf(parentSpan.Context()))
	sp.SetTag("name", name)
	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	callerDetails := fmt.Sprintf("%s - %s#%d", frame.Function, frame.File, frame.Line)
	sp.SetTag("caller", callerDetails)
	return sp
}
