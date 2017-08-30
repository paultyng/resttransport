// Package doctransport observes API registrations and traffic and generates
// Swagger. Ideally you could use this with unit tests which simulate API
// traffic.
package doctransport

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sync"

	"github.com/go-openapi/spec"
	"github.com/pkg/errors"

	"github.com/paultyng/resttransport"
	"github.com/paultyng/resttransport/routename"
)

type docTransport struct {
	sync.Mutex
	inner            resttransport.Transport
	spec             *spec.Swagger
	referenceStructs map[reflect.Type]bool
	namer            routename.Namer
}

type docRequestResponse struct {
	*docTransport
	inner resttransport.RequestResponse
	op    *spec.Operation
}

// SwaggerTransport is the interface for a Transport that has Swagger spec generation.
type SwaggerTransport interface {
	resttransport.Transport
	Generate() (*spec.Swagger, error)
}

// New returns an resttransport middleware that records interactions for documenting.
func New(inner resttransport.Transport) SwaggerTransport {
	return &docTransport{
		inner: inner,
		spec: &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				// TODO: swagger 3?
				Swagger: "2.0",
				// TODO: other types?
				Consumes: []string{"application/json"},
				Produces: []string{"application/json"},
			},
		},
		referenceStructs: map[reflect.Type]bool{},
		namer:            routename.New(),
	}
}

func (t *docTransport) Generate() (*spec.Swagger, error) {
	t.Lock()
	swagger := *t.spec
	t.Unlock()

	// TODO: support additional auth types
	swagger.SecurityDefinitions = spec.SecurityDefinitions{
		bearerTokenAuthorizationName: &spec.SecurityScheme{
			SecuritySchemeProps: spec.SecuritySchemeProps{
				Description: "Requires a bearer token",
				In:          "header",
				Name:        "Authorization",
				Type:        "apiKey",
			},
		},
	}

	if swagger.Definitions == nil {
		swagger.Definitions = spec.Definitions{}
	}

	for s := range t.referenceStructs {
		sch, err := schema(s, false)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to create schema for %v", s)
		}
		swagger.Definitions[s.Name()] = sch
	}

	return &swagger, nil
}

func addStructs(structs map[reflect.Type]bool, t reflect.Type) {
	switch t.Kind() {
	case reflect.Ptr,
		reflect.Slice,
		reflect.Array:
		addStructs(structs, t.Elem())
		return
	case reflect.Struct:
		if !structs[t] && t != wkTime {
			structs[t] = true
		}
	default:
		return
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		// assumes body only, so only json tags
		tag := f.Tag.Get("json")
		if isExportedField(f, tag) && !f.Anonymous {
			addStructs(structs, f.Type)
		}
	}
}

func (t *docTransport) addReferenceStruct(ref reflect.Type) {
	t.Lock()
	defer t.Unlock()

	addStructs(t.referenceStructs, ref)
}

func getOperation(pi spec.PathItem, httpMethod string) *spec.Operation {
	switch httpMethod {
	case "DELETE":
		return pi.Delete
	case "GET":
		return pi.Get
	case "HEAD":
		return pi.Head
	case "OPTIONS":
		return pi.Options
	case "PATCH":
		return pi.Patch
	case "POST":
		return pi.Post
	case "PUT":
		return pi.Put
	default:
		panic("unexpected http method " + httpMethod)
	}
}

func setOperation(pi *spec.PathItem, httpMethod string, op *spec.Operation) {
	switch httpMethod {
	case "DELETE":
		pi.Delete = op
	case "GET":
		pi.Get = op
	case "HEAD":
		pi.Head = op
	case "OPTIONS":
		pi.Options = op
	case "PATCH":
		pi.Patch = op
	case "POST":
		pi.Post = op
	case "PUT":
		pi.Put = op
	default:
		panic("unexpected http method " + httpMethod)
	}
}

func (t *docTransport) wrapHandler(auth bool, httpMethod, path string, inner resttransport.Handler) resttransport.Handler {
	t.Lock()
	defer t.Unlock()

	if t.spec.Paths == nil {
		t.spec.Paths = &spec.Paths{
			Paths: map[string]spec.PathItem{},
		}
	}

	id := t.namer.Name(httpMethod, path)
	pi := t.spec.Paths.Paths[path]
	op := getOperation(pi, httpMethod)

	if op == nil {
		op = &spec.Operation{
			OperationProps: spec.OperationProps{
				ID:          id,
				Description: "",
				Parameters:  []spec.Parameter{},
				Responses: &spec.Responses{
					ResponsesProps: spec.ResponsesProps{
						StatusCodeResponses: map[int]spec.Response{},
					},
				},
			},
		}
		if auth {
			op.Security = append(op.OperationProps.Security, map[string][]string{"Bearer": []string{}})
			op.Responses.StatusCodeResponses[http.StatusUnauthorized] = spec.Response{
				ResponseProps: spec.ResponseProps{
					Description: http.StatusText(http.StatusUnauthorized),
				},
			}
		}
		setOperation(&pi, httpMethod, op)
	}

	t.spec.Paths.Paths[path] = pi

	return func(ctx context.Context, reqres resttransport.RequestResponse) error {
		wrapper := &docRequestResponse{
			inner:        reqres,
			op:           op,
			docTransport: t,
		}

		return inner(ctx, wrapper)
	}
}

func (t *docTransport) RegisterHandler(httpMethod, path string, h resttransport.Handler) error {
	return t.inner.RegisterHandler(httpMethod, path, t.wrapHandler(false, httpMethod, path, h))
}

func (t *docTransport) RegisterAuthenticatedHandler(httpMethod, path string, h resttransport.Handler) error {
	return t.inner.RegisterAuthenticatedHandler(httpMethod, path, t.wrapHandler(true, httpMethod, path, h))
}

func (reqres *docRequestResponse) hasBodyParameter() bool {
	reqres.Lock()
	defer reqres.Unlock()
	for _, p := range reqres.op.Parameters {
		if p.In == "body" {
			return true
		}
	}
	return false
}

func (reqres *docRequestResponse) appendBodyParameter(v interface{}) error {
	const in = "body"

	if reqres.hasBodyParameter() {
		//short circuit if body param exists
		return nil
	}

	t := reflect.TypeOf(v)
	reqres.addReferenceStruct(t)

	typeSchema, err := schema(t, true)
	if err != nil {
		return errors.Wrap(err, "unable to map type for schema")
	}

	reqres.Lock()
	defer reqres.Unlock()
	reqres.op.Parameters = append(reqres.op.Parameters, spec.Parameter{
		ParamProps: spec.ParamProps{
			In:          in,
			Name:        in,
			Description: "",
			Required:    true,
			Schema:      &typeSchema,
		},
	})
	return nil
}

func (reqres *docRequestResponse) RequestHeader() http.Header {
	return reqres.inner.RequestHeader()
}

func (reqres *docRequestResponse) BindBody(v interface{}) error {
	err := reqres.appendBodyParameter(v)
	if err != nil {
		return errors.Wrap(err, "unable to append body parameter")
	}
	return reqres.inner.BindBody(v)
}

func (reqres *docRequestResponse) BindQuery(v interface{}) error {
	const in = "query"
	err := reqres.appendSimpleSchemaParameters(in, v)
	if err != nil {
		return err
	}
	return reqres.inner.BindQuery(v)
}

func (reqres *docRequestResponse) BindPath(v interface{}) error {
	const in = "path"
	err := reqres.appendSimpleSchemaParameters(in, v)
	if err != nil {
		return err
	}
	return reqres.inner.BindPath(v)
}

func (reqres *docRequestResponse) appendSimpleSchemaParameters(in string, v interface{}) error {
	reqres.Lock()
	defer reqres.Unlock()

	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("type %s in %s is not a struct", t.Name(), in)
	}
	return eachStructField(t, in, func(name string, required bool, s spec.Schema) error {
		st := ""
		if len(s.Type) > 0 {
			st = s.Type[0]
		}
		for _, p := range reqres.op.Parameters {
			if p.Name == name {
				//already exists
				return nil
			}
		}
		reqres.op.Parameters = append(reqres.op.Parameters, spec.Parameter{
			SimpleSchema: spec.SimpleSchema{
				Type:    st,
				Format:  s.Format,
				Default: s.Default,
			},
			ParamProps: spec.ParamProps{
				In:          in,
				Name:        name,
				Description: s.Description,
				Required:    required,
			},
		})

		return nil
	})
}

func (reqres *docRequestResponse) User() interface{} {
	return reqres.inner.User()
}

func (reqres *docRequestResponse) Body(status int, v interface{}) error {
	t := reflect.TypeOf(v)

	reqres.addReferenceStruct(t)

	typeSchema, err := schema(t, true)
	if err != nil {
		return errors.Wrap(err, "unable to map type for operation request")
	}

	resp := spec.Response{
		ResponseProps: spec.ResponseProps{
			Description: http.StatusText(status),
			Schema:      &typeSchema,
		},
	}

	reqres.Lock()
	defer reqres.Unlock()
	reqres.op.Responses.StatusCodeResponses[status] = resp

	return reqres.inner.Body(status, v)
}

func (reqres *docRequestResponse) NoBody(status int) error {
	resp := spec.Response{
		ResponseProps: spec.ResponseProps{
			Description: http.StatusText(status),
		},
	}

	reqres.Lock()
	defer reqres.Unlock()
	reqres.op.Responses.StatusCodeResponses[status] = resp

	return reqres.inner.NoBody(status)
}
