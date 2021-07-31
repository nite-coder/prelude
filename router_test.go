package prelude

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRoute(t *testing.T, path string) {
	passed := false

	router := newRouter()

	router.AddRoute(path, func(c *Context) error {
		passed = true
		return nil
	})

	cmd := Command{}
	c := NewContext(nil, &cmd)
	h := router.Find(path)
	err := h(c)

	require.NoError(t, err)
	assert.True(t, passed)
}

func TestRouterStaticRoute(t *testing.T) {
	testRoute(t, "/")
	testRoute(t, "/hello")
	testRoute(t, "/hello")
	testRoute(t, "/hello/put")
	testRoute(t, "/hello/Delet")
}
