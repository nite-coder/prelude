package prelude

type Context struct {
	SessionID string
}

func NewContext() *Context {
	return &Context{}
}
