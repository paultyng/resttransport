package doctransport

import (
	"mime/multipart"
	"reflect"
	"time"

	"github.com/go-openapi/jsonreference"
	"github.com/go-openapi/spec"
	"github.com/pkg/errors"
)

const bearerTokenAuthorizationName = "Bearer"

// well known types
var (
	wkTime                = reflect.TypeOf(time.Time{})
	wkMultipartFileHeader = reflect.TypeOf(multipart.FileHeader{})
)

func primitiveSchema(jsonSchemaType, format string) (spec.Schema, error) {
	if jsonSchemaType == "" {
		return spec.Schema{}, errors.Errorf("type is required")
	}

	return spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type:   []string{jsonSchemaType},
			Format: format,
		},
	}, nil
}

func isExportedField(f reflect.StructField, tag string) bool {
	return f.PkgPath == "" && tag != "-"
}

func eachStructField(t reflect.Type, tagName string, cb func(name string, required bool, s spec.Schema) error) error {
	var walkType func(t reflect.Type) error
	walkType = func(t reflect.Type) error {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Anonymous {
				// Anonymous structs should be flattened.
				err := walkType(f.Type)
				if err != nil {
					return err
				}
				continue
			}
			tag := f.Tag.Get(tagName)
			if isExportedField(f, tag) {
				name, _ := parseTag(tag)
				// ignoring name validity check here...
				if name == "" {
					name = f.Name
				}

				sch, err := schema(f.Type, true)
				if err != nil {
					return errors.Wrapf(err, "unable to get schema for field %s", f.Name)
				}

				err = cb(name, f.Type.Kind() != reflect.Ptr, sch)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walkType(t)
}

func structSchema(t reflect.Type) (spec.Schema, error) {
	if t.Kind() != reflect.Struct {
		return spec.Schema{}, errors.Errorf("%s is not a struct", t.Name())
	}

	props := map[string]spec.Schema{}
	required := []string{}
	err := eachStructField(t, "json", func(n string, r bool, s spec.Schema) error {
		props[n] = s
		if r {
			required = append(required, n)
		}
		return nil
	})
	if err != nil {
		return spec.Schema{}, errors.Wrapf(err, "unable to get properties for type %s", t.Name())
	}

	s := spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type:       []string{"object"},
			Properties: props,
			Required:   required,
		},
		SwaggerSchemaProps: spec.SwaggerSchemaProps{},
	}
	return s, nil
}

func refSchema(t reflect.Type) (spec.Schema, error) {
	if t.Kind() != reflect.Struct {
		return spec.Schema{}, errors.Errorf("%s is not a struct", t.Name())
	}

	ref, err := jsonreference.New("#/definitions/" + t.Name())
	if err != nil {
		return spec.Schema{}, errors.Wrap(err, "unable to create json ref")
	}
	return spec.Schema{
		SchemaProps: spec.SchemaProps{
			Ref: spec.Ref{
				Ref: ref,
			},
		},
	}, nil
}

func arraySchema(t reflect.Type, ref bool) (spec.Schema, error) {
	if t.Kind() != reflect.Struct && t.Kind() != reflect.Slice {
		return spec.Schema{}, errors.Errorf("%s is not an array", t.Name())
	}

	items, err := schema(t.Elem(), ref)
	if err != nil {
		return spec.Schema{}, errors.Wrapf(err, "unable to determine items for %s", t.Name())
	}

	return spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: []string{"array"},
			Items: &spec.SchemaOrArray{
				Schema: &items,
			},
		},
	}, nil
}

// nolint: gocyclo
func schema(t reflect.Type, ref bool) (spec.Schema, error) {
	switch t.Kind() {
	case reflect.Bool:
		return primitiveSchema("boolean", "")
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Int32:
		return primitiveSchema("integer", "int32")
	case reflect.Uint64,
		reflect.Int64:
		return primitiveSchema("integer", "int64")
	case reflect.Float32:
		return primitiveSchema("number", "float")
	case reflect.Float64:
		return primitiveSchema("number", "double")
	case reflect.String:
		//TODO: formats?
		return primitiveSchema("string", "")
	case reflect.Ptr:
		return schema(t.Elem(), ref)
	case reflect.Array,
		reflect.Slice:
		return arraySchema(t, ref)
	case reflect.Struct:
		switch t {
		case wkTime:
			return primitiveSchema("string", "date-time")
		case wkMultipartFileHeader:
			return primitiveSchema("file", "")
		}
		if ref {
			return refSchema(t)
		}
		return structSchema(t)
	}
	return spec.Schema{}, errors.Errorf("unexpected kind %v", t.Kind())
}
