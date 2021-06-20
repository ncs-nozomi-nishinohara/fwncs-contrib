package trace

import (
	"net/http"

	"github.com/n-creativesystem/go-fwncs"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/stacktrace"
)

const componentName = "fwncs/v0"

type Config struct {
	Tracer        *apm.Tracer
	ComponentName string
}

func Tracing(tracer *apm.Tracer) fwncs.HandlerFunc {
	return TracingWithConfig(Config{
		Tracer:        tracer,
		ComponentName: componentName,
	})
}

func TracingWithConfig(config Config) fwncs.HandlerFunc {
	stacktrace.RegisterLibraryPackage(config.ComponentName)
	return func(c fwncs.Context) {
		w := c.Writer()
		req := c.Request()
		requestName := c.Method() + " " + c.Path()
		tx, body, r := apmhttp.StartTransactionWithBody(config.Tracer, requestName, req)
		defer tx.End()
		*req = *r
		defer func() {
			if v := recover(); v != nil {
				w.WriteHeader(http.StatusInternalServerError)
				ec := config.Tracer.Recovered(v)
				ec.SetTransaction(tx)
				setElasticContext(&ec.Context, req, w, body)
				ec.Send()
			}
			tx.Result = apmhttp.StatusCodeResult(w.Status())
			if tx.Sampled() {
				setElasticContext(&tx.Context, req, w, body)
			}
			body.Discard()
		}()
		c.Next()
	}
}

func setElasticContext(ctx *apm.Context, req *http.Request, w fwncs.ResponseWriter, body *apm.BodyCapturer) {
	ctx.SetFramework("fwncs", "v0")
	ctx.SetHTTPRequest(req)
	ctx.SetHTTPRequestBody(body)
	ctx.SetHTTPStatusCode(w.Status())
	ctx.SetHTTPResponseHeaders(w.Header())
}
