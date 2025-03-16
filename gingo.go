package gingo

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Engine struct {
	*gin.Engine
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
		Engine: gin.New(),
	}
	return e.With(opts...)
}

// Default returns an Engine instance with the Logger and Recovery middleware already attached.
func Default(opts ...OptionFunc) *Engine {
	e := &Engine{
		Engine: gin.Default(),
	}
	return e.With(opts...)
}

func (e *Engine) Handle(httpMethod, relativePath string, handlers ...HandlerFunc) {
	// Convert your handlers to gin.HandlerFunc
	ginHandlers := make([]gin.HandlerFunc, len(handlers))
	for _, handler := range handlers {
		h := handler // create a copy to avoid closure issues
		ginHandlers = append(
			ginHandlers,
			func(ginContext *gin.Context) {
				context := &Context{Context: ginContext}
				h(context)
			},
		)
	}

	// Register with gin
	e.Engine.Handle(httpMethod, relativePath, ginHandlers...)
}

// Common HTTP method shortcuts
func (r *Engine) GET(relativePath string, handlers ...HandlerFunc) {
	r.Handle(http.MethodGet, relativePath, handlers...)
}

func (r *Engine) POST(relativePath string, handlers ...HandlerFunc) {
	r.Handle(http.MethodPost, relativePath, handlers...)
}

func (r *Engine) PUT(relativePath string, handlers ...HandlerFunc) {
	r.Handle(http.MethodPut, relativePath, handlers...)
}

func (r *Engine) PATCH(relativePath string, handlers ...HandlerFunc) {
	r.Handle(http.MethodPatch, relativePath, handlers...)
}

func (r *Engine) OPTIONS(relativePath string, handlers ...HandlerFunc) {
	r.Handle(http.MethodOptions, relativePath, handlers...)
}
func (r *Engine) DELETE(relativePath string, handlers ...HandlerFunc) {
	r.Handle(http.MethodDelete, relativePath, handlers...)
}
