package resttransport

// Transport represents the mapping between an API and the underlying communication infrastructure.
// Handlers can be registered with or without authentication. URL variable annotations should follow
// the form `/foo/{id}` where brackets are used to denote path parameters.
type Transport interface {
	RegisterHandler(httpMethod, path string, h Handler) error
	RegisterAuthenticatedHandler(httpMethod, path string, h Handler) error
}

// RequestResponse represents the handlers contract for reading request data and responding. Both
// BindQuery and BindPath assume a struct (can be anonymous), optionally including struct field
// tags, to bind to variables extracted from the requested URL. Implementation of the annotations
// and URL registration is specific to transport implementation. Similar to JSON marshaling, fields
// must be exported to be bound.
//
// If a handler registered as `/foos/{id}/bars` has a request URL similar to `/foos/1/bars?page=3`,
// this could be bound similar to the following:
//
//	func getFooBars(r RequestResponse) error {
//		pathParams := struct{ ID string `path:"id"` }{}
//		r.BindPath(&pathParams)
//
//		queryParams := struct{ Page string `path:"page"` }{}
//		r.BindQuery(&queryParams)
//
//		// use pathParams.ID, queryParams.Page, etc...
//	}
type RequestResponse interface {
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

// Handler represents a func that processes a RequestResponse.
type Handler func(RequestResponse) error
