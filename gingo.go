package gingo

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type Engine struct {
	*gin.Engine
	definitions []RouteDefinition
	hookManager HookManager
	pool        sync.Pool
}

type OptionFunc func(*Engine)

func (engine *Engine) With(opts ...OptionFunc) *Engine {
	for _, opt := range opts {
		o := opt // copy to avoid closure issues
		o(engine)
	}
	return engine
}

type RouterGroup struct {
	*gin.RouterGroup
}

type (
	HandlerFunc  func(*Context)
	HandlerChain []HandlerFunc
)

type RouteDefinition struct {
	Method                string
	Path                  string
	RequestHandler        HandlerFunc
	PreRequestMiddleware  HandlerChain
	PostRequestMiddleware HandlerChain
	Description           string
	Tags                  []string
}

func (r RouteDefinition) getHandlers() HandlerChain {
	temp := append(r.PreRequestMiddleware, r.RequestHandler)
	return append(temp, r.PostRequestMiddleware...)
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
		pool: sync.Pool{New: func() interface{} {
			return &Context{Context: nil}
		}},
	}
	e.Use(e.getBootstrapHooks()...)
	return e.With(opts...)
}

// Default returns an Engine instance with the Logger and Recovery middleware already attached.
func Default(opts ...OptionFunc) *Engine {
	e := New()
	e.Engine.Use(gin.Logger(), gin.Recovery())
	return e.With(opts...)
}

func (engine *Engine) Handle(httpMethod string, relativePath string, handlers ...HandlerFunc) {
	routeDef := &RouteDefinition{
		Method:                httpMethod,
		Path:                  relativePath,
		Description:           fmt.Sprintf("%s: %s", strings.ToUpper(httpMethod), relativePath),
		RequestHandler:        handlers[len(handlers)-1],
		PreRequestMiddleware:  handlers[:len(handlers)-1],
		PostRequestMiddleware: make(HandlerChain, 0),
	}

	engine.handle(routeDef)
}

func (engine *Engine) getBootstrapHooks() []gin.HandlerFunc {
	return gin.HandlersChain{
		func(ginContext *gin.Context) {
			// Get context from pool
			c := engine.pool.Get().(*Context)

			c.Context = ginContext
			c.engine = engine
			c.Reset()

			// Store in gin context
			ginContext.Set("gingo_context", c)

			// Process request
			ginContext.Next()

			// Return to pool
			engine.pool.Put(c)
		},
	}
}

func (engine *Engine) handle(routeDef *RouteDefinition) {
	ginHandlers := engine.mapHandlers(routeDef)

	// Register with gin
	engine.Engine.Handle(routeDef.Method, routeDef.Path, ginHandlers...)
}

func (engine *Engine) GET(relativePath string, handlers ...HandlerFunc) {
	engine.Handle(http.MethodGet, relativePath, handlers...)
}

func (engine *Engine) POST(relativePath string, handlers ...HandlerFunc) {
	engine.Handle(http.MethodPost, relativePath, handlers...)
}

func (engine *Engine) PUT(relativePath string, handlers ...HandlerFunc) {
	engine.Handle(http.MethodPut, relativePath, handlers...)
}

func (engine *Engine) PATCH(relativePath string, handlers ...HandlerFunc) {
	engine.Handle(http.MethodPatch, relativePath, handlers...)
}

func (engine *Engine) OPTIONS(relativePath string, handlers ...HandlerFunc) {
	engine.Handle(http.MethodOptions, relativePath, handlers...)
}

func (engine *Engine) DELETE(relativePath string, handlers ...HandlerFunc) {
	engine.Handle(http.MethodDelete, relativePath, handlers...)
}

// executeHooks runs all hooks for a specific hook type
func (engine *Engine) executeHooks(
	hookType HookType,
	ginContext *gin.Context,
	additionalHooks ...HandlerFunc,
) {
	contextValue, ok := ginContext.Get("gingo_context")
	if !ok {
		panic("gingo context does not exist")
	}
	c := contextValue.(*Context)

	// Runs predefined global hooks, along with additional request middleware
	hooks := append(engine.hookManager.hooks[hookType], additionalHooks...)
	for _, hook := range hooks {
		h := hook // Copy to avoid closure issues
		h(c)

		// Check if response was aborted
		if ginContext.IsAborted() {
			break
		}
	}
}

func (engine *Engine) mapHandlers(routeDef *RouteDefinition) gin.HandlersChain {
	var ginHandlers gin.HandlersChain

	// First middleware to set the route definition
	ginHandlers = append(ginHandlers, func(ginContext *gin.Context) {
		contextValue, exists := ginContext.Get("gingo_context")
		if !exists {
			panic("Bootstrap middleware not called - gingo_context missing")
		}

		c := contextValue.(*Context)
		c.routeDef = routeDef // Store route definition in context
		ginContext.Next()
	})

	// Before request hooks
	if len(engine.hookManager.hooks[BeforeRequest])+len(routeDef.PreRequestMiddleware) > 0 {
		ginHandlers = append(ginHandlers, func(ginContext *gin.Context) {
			engine.executeHooks(BeforeRequest, ginContext, routeDef.PreRequestMiddleware...)
			ginContext.Next()
		})
	}

	// Main handler
	handler := routeDef.RequestHandler
	if handler == nil {
		panic(fmt.Sprintf("no request handler defined for %s", routeDef.Path))
	}

	ginHandlers = append(ginHandlers, func(ginContext *gin.Context) {
		contextValue, exists := ginContext.Get("gingo_context")
		if !exists {
			panic("gingo context does not exist")
		}
		c := contextValue.(*Context)
		handler(c)

		if !ginContext.IsAborted() {
			ginContext.Next()
		}
	})

	// After request hooks
	if len(engine.hookManager.hooks[AfterRequest])+len(routeDef.PostRequestMiddleware) > 0 {
		ginHandlers = append(ginHandlers, func(ginContext *gin.Context) {
			engine.executeHooks(AfterRequest, ginContext, routeDef.PostRequestMiddleware...)
		})
	}

	return ginHandlers
}
