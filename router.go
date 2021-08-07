package prelude

import "strings"

// HandlerFunc defines a function to server HTTP requests
type HandlerFunc func(c *Context) error

type Router struct {
	name string
	tree *tree
	hub  Huber
}

type tree struct {
	rootNode *node
}

type kind uint8

var (
	notFoundHandler = func(c *Context) {

	}
)

const (
	skind kind = iota
	pkind
	akind
)

// NewRouter function will create a new router instance
func NewRouter(name string, hub Huber) *Router {
	r := newRouter()
	r.hub = hub
	r.name = name
	hub.SetRouter(r)
	return r
}

func newRouter() *Router {
	r := Router{
		name: "default",
		tree: &tree{
			rootNode: &node{
				parent:    nil,
				children:  []*node{},
				kind:      0,
				name:      "",
				sortOrder: 0,
			},
		},
	}
	return &r
}

// AddRoute function which adding action and handler to router
func (r *Router) AddRoute(action string, handler HandlerFunc) {
	if len(action) == 0 {
		panic("router: action couldn't be empty")
	}

	currentNode := r.tree.rootNode
	if action == "" {
		currentNode.handler = handler
		return
	}

	actions := strings.Split(action, ".")
	count := len(actions)
	pathParams := []string{}

	for index, element := range actions {
		if len(element) == 0 {
			continue
		}

		var childNode *node
		// this is static node
		childNode = currentNode.findChildByName(element)
		if childNode == nil {
			childNode = newNode(element, skind)
			currentNode.addChild(childNode)

		}

		// last node in the path
		if count == index+1 {
			childNode.params = pathParams
			childNode.handler = handler
		}

		currentNode = childNode
	}

	if r.hub == nil {
		return
	}

	_ = r.hub.QueueSubscribe(action)
}

// Find returns http handler for specific path
func (r *Router) Find(path string) HandlerFunc {
	currentNode := r.tree.rootNode
	if path == "" {
		return currentNode.handler
	}

	pathArray := strings.Split(path, ".")
	count := len(pathArray)

	for index, element := range pathArray {
		// find static node first
		childNode := currentNode.findChildByName(element)

		if childNode == nil {
			// return notFoundHandler
			return nil
		}

		// last node in the path
		if count == index+1 {
			return childNode.handler
		}

		currentNode = childNode
	}

	// return notFoundHandler
	return nil
}

type node struct {
	parent    *node
	children  []*node
	kind      kind
	name      string
	pNames    []string
	params    []string
	sortOrder int
	handler   HandlerFunc
}

func newNode(name string, t kind) *node {
	return &node{
		kind:      t,
		name:      name,
		sortOrder: 0,
	}
}

func (n *node) addChild(node *node) {
	node.parent = n
	n.children = append(n.children, node)
}

func (n *node) findChildByName(name string) *node {
	var result *node
	for _, element := range n.children {
		if strings.EqualFold(element.name, name) && element.kind == skind {
			result = element
			break
		}
	}
	return result
}

func (n *node) findChildByKind(t kind) *node {
	for _, c := range n.children {
		if c.kind == t {
			return c
		}
	}
	return nil
}
