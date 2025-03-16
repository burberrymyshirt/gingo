package gingo

import "github.com/gin-gonic/gin"

type Engine struct {
	*gin.Engine
}

func (e *Engine) With(opts ...OptionFunc) *Engine {
	for _, opt := range opts {
		opt(e)
	}
	return e
}

type RouterGroup struct {
	*gin.RouterGroup
}

type HandlerFunc func(*Context)

type OptionFunc func(*Engine)

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
	r.Handle("GET", relativePath, handlers...)
}

func (r *Engine) POST(relativePath string, handlers ...HandlerFunc) {
	r.Handle("POST", relativePath, handlers...)
}
