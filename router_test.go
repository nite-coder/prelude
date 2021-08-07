package prelude

import (
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"

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

	event := cloudevents.NewEvent()
	c := NewContext(nil, event)
	h := router.Find(path)
	err := h(c)

	require.NoError(t, err)
	assert.True(t, passed)
}

func TestRouterStaticRoute(t *testing.T) {
	testRoute(t, "hello")
	testRoute(t, "hello.put")
	testRoute(t, "hello.Delet")
}
