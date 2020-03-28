package prelude

type Context struct {
	SessionID string
}

func NewContext() *Context {
	return &Context{}
}

func (c *Context) Get(name string) string {
	return ""
}

func (c *Context) WriteBytes(name string) string {
	return ""
}
