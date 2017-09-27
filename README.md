# resttransport
--
    import "github.com/paultyng/resttransport"

Package resttransport provides a simple abstraction for REST API's that allows
for simpler API implementation.

[![Build
Status](https://travis-ci.org/paultyng/resttransport.svg?branch=master)](https://travis-ci.org/paultyng/resttransport)

## Usage

#### type Handler

```go
type Handler func(context.Context, RequestResponse) error
```

Handler represents a func that processes a RequestResponse.

#### type RequestResponse

```go
type RequestResponse interface {
	RequestHeader() http.Header

	// BindQuery binds a struct to query string variables extracted from the requested URL.
	BindQuery(interface{}) error
	// BindBody binds the HTTP request body using the transports configured marshaling (which should
	// be based on HTTP request content type).
	BindBody(interface{}) error
	// BindPath binds a struct to path variables extracted from the requested URL.
	BindPath(interface{}) error

	// User returns current user state/context (differs based on transport implementations).
	User() interface{}

	// Body sends a response body with the given status code. The type of marshaling will be decided
	// by the transport.
	Body(status int, body interface{}) error
	// NoBody sends a response with only a status code.
	NoBody(status int) error
}
```

RequestResponse represents the handlers contract for reading request data and
responding. Both BindQuery and BindPath assume a struct (can be anonymous),
optionally including struct field tags, to bind to variables extracted from the
requested URL. Implementation of the annotations and URL registration is
specific to transport implementation. Similar to JSON marshaling, fields must be
exported to be bound.

If a handler registered as `/foos/{id}/bars` has a request URL similar to
`/foos/1/bars?page=3`, this could be bound similar to the following:

    func getFooBars(r RequestResponse) error {
    	pathParams := struct{ ID string `path:"id"` }{}
    	r.BindPath(&pathParams)

    	queryParams := struct{ Page string `path:"page"` }{}
    	r.BindQuery(&queryParams)

    	// use pathParams.ID, queryParams.Page, etc...
    }

#### type Transport

```go
type Transport interface {
	RegisterHandler(httpMethod, path string, h Handler) error
	RegisterAuthenticatedHandler(httpMethod, path string, h Handler) error
}
```

Transport represents the mapping between an API and the underlying communication
infrastructure. Handlers can be registered with or without authentication. URL
variable annotations should follow the form `/foo/{id}` where brackets are used
to denote path parameters.

# doctransport
--
    import "github.com/paultyng/resttransport/doctransport"

Package doctransport observes API registrations and traffic and generates
Swagger. Ideally you could use this with unit tests which simulate API traffic.

## Usage

#### type SwaggerTransport

```go
type SwaggerTransport interface {
	resttransport.Transport
	Generate() (*spec.Swagger, error)
}
```

SwaggerTransport is the interface for a Transport that has Swagger spec
generation.

#### func  New

```go
func New(inner resttransport.Transport) SwaggerTransport
```
New returns an resttransport middleware that records interactions for
documenting.

# echotransport
--
    import "github.com/paultyng/resttransport/echotransport"

Package echotransport wraps Echo for use as a resttransport implementation.

## Usage

#### func  New

```go
func New(c *Config) resttransport.Transport
```
New returns a Transport that wraps an Echo application.

#### type Config

```go
type Config struct {
	Echo                     EchoOrContext
	AuthenticationMiddleware []echo.MiddlewareFunc
	UserContextKey           string
}
```

Config holds configuration information for the Echo Transport.

#### type EchoOrContext

```go
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
```

EchoOrContext represents an Echo application struct or Context interface.
