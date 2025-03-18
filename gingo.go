package gingo

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Engine struct {
	*gin.Engine
	definitions []RouteDefinition
	hookManager HookManager
}

type OptionFunc func(*Engine)

func (e *Engine) With(opts ...OptionFunc) *Engine {
	for _, opt := range opts {
		o := opt // copy to avoid closure issues
		o(e)
	}
	return e
}

type RouterGroup struct {
	*gin.RouterGroup
}

type HandlerFunc func(*Context)

type RouteDefinition struct {
	Method                string
	Path                  string
	RequestHandler        HandlerFunc
	PreRequestMiddleware  []HandlerFunc
	PostRequestMiddleware []HandlerFunc
	Description           string
	Tags                  []string
}

// New returns a new blank Engine instance without any middleware attached.
// By default, the configuration is:
// - RedirectTrailingSlash:  true
// - RedirectFixedPath:      false
// - HandleMethodNotAllowed: false
// - ForwardedByClientIP:    true
// - UseRawPath:             false
// - UnescapePathValues:     true
func New(opts ...OptionFunc) *Engine {
	e := &Engine{
		Engine:      gin.New(),
		hookManager: *NewHookManager(),
	}
	return e.With(opts...)
}

// Default returns an Engine instance with the Logger and Recovery middleware already attached.
func Default(opts ...OptionFunc) *Engine {
	e := &Engine{
		Engine:      gin.Default(),
		hookManager: *NewHookManager(),
	}
	return e.With(opts...)
}

func (e *Engine) Handle(httpMethod string, relativePath string, handlers ...HandlerFunc) {
	routeDef := &RouteDefinition{
		Method:                httpMethod,
		Path:                  relativePath,
		Description:           fmt.Sprintf("%s: %s", strings.ToUpper(httpMethod), relativePath),
		RequestHandler:        handlers[len(handlers)-1],
		PreRequestMiddleware:  handlers[:len(handlers)-1],
		PostRequestMiddleware: make([]HandlerFunc, 0),
	}

	e.handle(routeDef)
}

func (e *Engine) handle(routeDef *RouteDefinition) {
	ginHandlers := e.hookManager.mapHandlers(routeDef)

	// Register with gin
	e.Engine.Handle(routeDef.Method, routeDef.Path, ginHandlers...)
}

func (e *Engine) GET(relativePath string, handlers ...HandlerFunc) {
	e.Handle(http.MethodGet, relativePath, handlers...)
}

func (e *Engine) POST(relativePath string, handlers ...HandlerFunc) {
	e.Handle(http.MethodPost, relativePath, handlers...)
}

func (e *Engine) PUT(relativePath string, handlers ...HandlerFunc) {
	e.Handle(http.MethodPut, relativePath, handlers...)
}

func (e *Engine) PATCH(relativePath string, handlers ...HandlerFunc) {
	e.Handle(http.MethodPatch, relativePath, handlers...)
}

func (e *Engine) OPTIONS(relativePath string, handlers ...HandlerFunc) {
	e.Handle(http.MethodOptions, relativePath, handlers...)
}

func (e *Engine) DELETE(relativePath string, handlers ...HandlerFunc) {
	e.Handle(http.MethodDelete, relativePath, handlers...)
}
