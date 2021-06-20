package trace

import (
	"context"
	"net/http"

	"github.com/n-creativesystem/go-fwncs"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

// func init() {
// 	internal.TrackUsage("integration", "framework", PackageName, Version)
// }

const NewRelicAppKey = "newrelicApp"

type newrelicResponseWriter struct {
	fwncs.ResponseWriter
	replacement http.ResponseWriter
	code        int
	written     bool
}

var _ fwncs.ResponseWriter = &newrelicResponseWriter{}

func (w *newrelicResponseWriter) flushHeader() {
	if !w.written {
		w.replacement.WriteHeader(w.code)
		w.written = true
	}
}

func (w *newrelicResponseWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *newrelicResponseWriter) Write(data []byte) (int, error) {
	w.flushHeader()
	return w.ResponseWriter.Write(data)
}

func (w *newrelicResponseWriter) WriteString(s string) (int, error) {
	w.flushHeader()
	return w.ResponseWriter.WriteString(s)
}

func (w *newrelicResponseWriter) WriteHeaderNow() {
	w.flushHeader()
	w.ResponseWriter.WriteHeaderNow()
}

const componentName = "fwncs/v0"

type Config struct {
	Application   *newrelic.Application
	ComponentName string
}

func Tracing(app *newrelic.Application) fwncs.HandlerFunc {
	return TracingWithConfig(Config{
		Application:   app,
		ComponentName: componentName,
	})
}

func TracingWithConfig(config Config) fwncs.HandlerFunc {
	if config.Application == nil {
		panic("trace middleware requires newrelic application")
	}
	if config.ComponentName == "" {
		config.ComponentName = componentName
	}

	return func(c fwncs.Context) {
		w := c.Writer()
		req := c.Request()
		requestName := c.Method() + " " + c.Path()
		tx := config.Application.StartTransaction(requestName)
		tx.SetWebRequestHTTP(req)
		defer tx.End()
		repl := &newrelicResponseWriter{
			ResponseWriter: w,
			replacement:    tx.SetWebResponse(w),
			code:           http.StatusOK,
		}
		w = repl
		defer repl.flushHeader()
		*req = *req.WithContext(context.WithValue(req.Context(), NewRelicAppKey, tx))
		c.SetRequest(req)
		c.SetWriter(w)
		c.Next()
	}
}
