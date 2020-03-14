package prelude

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testRoute(t *testing.T, cmd string) {
	passed := false

	router := NewRouter()
	router.Add(cmd, func(c *Context) error {
		passed = true
		return nil
	})

	c := NewContext()
	h := router.Find(cmd)
	err := h(c)
	if err == nil {

	}

	assert.True(t, passed)
}

func TestRouterStaticRoute(t *testing.T) {
	testRoute(t, "/")
	testRoute(t, "/hello")
	testRoute(t, "/hello")
	testRoute(t, "/hello/put")
	testRoute(t, "/hello/Delet")
}
