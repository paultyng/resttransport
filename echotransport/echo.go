// Package echotransport wraps Echo for use as a resttransport implementation.
package echotransport

import (
	"mime/multipart"
	"net/http"

	"github.com/gorilla/schema"
	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/paultyng/resttransport"
)

var (
	pathDecoder  = schema.NewDecoder()
	queryDecoder = schema.NewDecoder()
)

func init() {
	pathDecoder.SetAliasTag("path")
	queryDecoder.SetAliasTag("query")
}

type echoTransport struct {
	echo                     EchoOrContext
	authenticationMiddleware []echo.MiddlewareFunc
	userKey                  string
}

// EchoOrContext represents an Echo application struct or Context interface.
type EchoOrContext interface {
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

// Config holds configuration information for the Echo Transport.
type Config struct {
	Echo                     EchoOrContext
	AuthenticationMiddleware []echo.MiddlewareFunc
	UserContextKey           string
}

// New returns a Transport that wraps an Echo application.
func New(c *Config) resttransport.Transport {
	if c == nil {
		c = &Config{
			UserContextKey: "user",
		}
	}
	e := c.Echo
	if e == nil {
		e = echo.New()
	}
	return &echoTransport{
		echo: e,
		authenticationMiddleware: c.AuthenticationMiddleware,
		userKey:                  c.UserContextKey,
	}
}

type echoRequestResponse struct {
	userKey string
	c       echo.Context
}

func (rr *echoRequestResponse) RequestHeader() http.Header {
	return rr.c.Request().Header
}

func (rr *echoRequestResponse) BindQuery(v interface{}) error {
	q := rr.c.QueryParams()
	return queryDecoder.Decode(v, q)
}

func (rr *echoRequestResponse) BindBody(v interface{}) error {
	return rr.c.Bind(v)
}

func (rr *echoRequestResponse) BindPath(v interface{}) error {
	values := map[string][]string{}
	for _, n := range rr.c.ParamNames() {
		values[n] = []string{rr.c.Param(n)}
	}
	return pathDecoder.Decode(v, values)
}

func (rr *echoRequestResponse) Body(status int, body interface{}) error {
	// TODO: based on accepted content types?
	return rr.c.JSON(status, body)
}

func (rr *echoRequestResponse) NoBody(status int) error {
	return rr.c.NoContent(status)
}

func (rr *echoRequestResponse) Attachment(file, name, contentType string) error {
	rr.c.Response().Header().Set("Content-Type", contentType)
	return rr.c.Attachment(file, name)
}

func (rr *echoRequestResponse) User() interface{} {
	return rr.c.Get(rr.userKey)
}

func (rr *echoRequestResponse) FormFile(name string) (*multipart.FileHeader, error) {
	return rr.c.FormFile(name)
}

func (t *echoTransport) echoHandlerWrapper(h resttransport.Handler) echo.HandlerFunc {
	return func(c echo.Context) error {
		reqresp := &echoRequestResponse{
			c:       c,
			userKey: t.userKey,
		}
		return h(c.Request().Context(), reqresp)
	}
}

func (t *echoTransport) RegisterHandler(httpMethod, path string, consumes []string, h resttransport.Handler) error {
	return t.register(httpMethod, replacePathParameters(path), h)
}

func (t *echoTransport) RegisterAuthenticatedHandler(httpMethod, path string, consumes []string, h resttransport.Handler) error {
	return t.register(httpMethod, replacePathParameters(path), h, t.authenticationMiddleware...)
}

func (t *echoTransport) register(httpMethod, path string, h resttransport.Handler, mw ...echo.MiddlewareFunc) error {
	var reg func(string, echo.HandlerFunc, ...echo.MiddlewareFunc) *echo.Route

	switch httpMethod {
	case "GET":
		reg = t.echo.GET
	case "HEAD":
		reg = t.echo.HEAD
	case "OPTIONS":
		reg = t.echo.OPTIONS
	case "PATCH":
		reg = t.echo.PATCH
	case "POST":
		reg = t.echo.POST
	case "PUT":
		reg = t.echo.PUT
	case "TRACE":
		reg = t.echo.TRACE
	case "DELETE":
		reg = t.echo.DELETE
	default:
		return errors.Errorf("unexpected method '%s'", httpMethod)
	}

	reg(path, t.echoHandlerWrapper(h), mw...)
	return nil
}
