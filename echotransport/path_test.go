package echotransport

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplacePathParamters(t *testing.T) {
	for i, c := range []struct {
		expected string
		path     string
	}{
		{"/", "/"},
		{"", ""},
		{"/foo", "/foo"},
		{"/foo/:bar", "/foo/{bar}"},
		{"/foo/:bar/baz", "/foo/{bar}/baz"},
	} {
		t.Run(fmt.Sprintf("%d %s", i, c.path), func(t *testing.T) {
			assert := assert.New(t)

			actual := replacePathParameters(c.path)
			assert.Equal(c.expected, actual)
		})
	}
}
