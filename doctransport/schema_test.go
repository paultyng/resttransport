package doctransport

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStructSchema(t *testing.T) {
	assert := require.New(t)

	type Category struct {
		ID            int64 `json:"id"`
		Name          string
		NullableThing *string `json:"nt"`
		ArrayOfThings []int   `json:"aot"`
	}

	schema, err := structSchema(reflect.TypeOf(Category{}))
	assert.NoError(err)
	assert.Equal(spec.StringOrArray([]string{"object"}), schema.SchemaProps.Type)
	assert.Equal(4, len(schema.SchemaProps.Properties))
	assert.Equal(3, len(schema.SchemaProps.Required))
	assert.Contains(schema.SchemaProps.Required, "id")
	assert.Contains(schema.SchemaProps.Required, "Name")
	assert.Contains(schema.SchemaProps.Required, "aot")
	assert.NotContains(schema.SchemaProps.Required, "nt")

	id, ok := schema.SchemaProps.Properties["id"]
	assert.True(ok)
	assert.Equal(spec.StringOrArray([]string{"integer"}), id.SchemaProps.Type)
	assert.Equal("int64", id.SchemaProps.Format)

	name, ok := schema.SchemaProps.Properties["Name"]
	assert.True(ok)
	assert.Equal(spec.StringOrArray([]string{"string"}), name.SchemaProps.Type)

	nullableThing, ok := schema.SchemaProps.Properties["nt"]
	assert.True(ok)
	assert.Equal(spec.StringOrArray([]string{"string"}), nullableThing.SchemaProps.Type)

	arrayOfThings, ok := schema.SchemaProps.Properties["aot"]
	assert.True(ok)
	assert.Equal(spec.StringOrArray([]string{"array"}), arrayOfThings.SchemaProps.Type)
	assert.NotNil(arrayOfThings.SchemaProps.Items)
	items := arrayOfThings.SchemaProps.Items.Schema
	if items == nil {
		items = &arrayOfThings.SchemaProps.Items.Schemas[0]
	}
	assert.NotNil(items)
	assert.Equal(spec.StringOrArray([]string{"integer"}), items.SchemaProps.Type)
	assert.Equal("int32", items.SchemaProps.Format)
}

func TestStructSchema_AnonymousStructs(t *testing.T) {
	assert := require.New(t)
	type Category struct {
		ID int64 `json:"id"`
	}

	type CategoryImplementer struct {
		Category
		AdditionalThing string `json:"additionalThing"`
	}

	type CategoryImplementerImplementer struct {
		CategoryImplementer
		AnotherAdditionalThing *string `json:"anotherAdditionalThing"`
	}
	schema, err := structSchema(reflect.TypeOf(CategoryImplementerImplementer{}))
	assert.NoError(err)
	assert.Equal(spec.StringOrArray([]string{"object"}), schema.SchemaProps.Type)
	assert.Equal(3, len(schema.SchemaProps.Properties))
	assert.Equal(2, len(schema.SchemaProps.Required))
	assert.Contains(schema.SchemaProps.Required, "id")
	// This comes from an anonymous struct
	assert.Contains(schema.SchemaProps.Required, "additionalThing")
	// This comes from another anonymous struct, nested and not required
	assert.NotContains(schema.SchemaProps.Required, "anotherAdditionalThing")
}

func TestSchema(t *testing.T) {
	cases := []struct {
		expectedType   string
		expectedFormat string
		t              reflect.Type
	}{
		{"boolean", "", reflect.TypeOf(false)},
		{"integer", "int32", reflect.TypeOf(int(0))},
		{"integer", "int32", reflect.TypeOf(int8(0))},
		{"integer", "int32", reflect.TypeOf(int16(0))},
		{"integer", "int32", reflect.TypeOf(int32(0))},
		{"integer", "int32", reflect.TypeOf(uint(0))},
		{"integer", "int32", reflect.TypeOf(uint8(0))},
		{"integer", "int32", reflect.TypeOf(uint16(0))},
		{"integer", "int32", reflect.TypeOf(uint32(0))},
		{"integer", "int64", reflect.TypeOf(int64(0))},
		{"integer", "int64", reflect.TypeOf(uint64(0))},
		{"number", "float", reflect.TypeOf(float32(0))},
		{"number", "double", reflect.TypeOf(float64(0))},
		{"string", "", reflect.TypeOf(string(0))},
		{"string", "date-time", reflect.TypeOf(time.Time{})},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("%d %s %s", i, c.expectedType, c.expectedFormat), func(t *testing.T) {
			assert := assert.New(t)
			sch, err := schema(c.t, false)
			assert.NoError(err)
			assert.Equal(spec.StringOrArray([]string{c.expectedType}), sch.SchemaProps.Type)
			assert.Equal(c.expectedFormat, sch.SchemaProps.Format)
		})
	}
}

// func TestAddStructs(t *testing.T) {
// 	assert := assert.New(t)

// 	type Toy struct {
// 		Name      string
// 		CreatedAt time.Time
// 	}

// 	type Child struct {
// 		FavoriteToy *Toy
// 	}

// 	type Parent struct {
// 		Children []Child
// 	}

// 	structs := map[reflect.Type]bool{}
// 	addStructs(structs, reflect.TypeOf((*Parent)(nil)))

// 	assert.Equal(3, len(structs))
// 	assert.True(structs[reflect.TypeOf(Parent{})], "has Parent")
// 	assert.True(structs[reflect.TypeOf(Toy{})], "has Toy")
// 	assert.True(structs[reflect.TypeOf(Child{})], "has Child")
// 	assert.False(structs[reflect.TypeOf(time.Time{})], "does not have time.Time")
// }
