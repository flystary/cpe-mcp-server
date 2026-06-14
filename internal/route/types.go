package route

import (
	"context"
	"encoding/json"
)

type Route struct {
	CIDR     string
	NextHop  string
	Iface    string
	Protocol string // static / bgp / ospf
	Metric   int
}

type Base interface {
	Name() string
}
type Actioner interface {
	Actions() []string
}

type Controller interface {
	Dispatch(ctx context.Context, action string, args json.RawMessage) error
}

type Reader interface {
	List(ctx context.Context) (interface{}, error)
}

type SchemaProvider interface {
	Schema() Schema
}

type Validator interface {
	Validate(action string, args []byte) (map[string]any, error)
}

type RouteHandler interface {
	Base
	Actioner
	Controller
	Reader
	SchemaProvider
	Validator
}

type Factory func() RouteHandler
