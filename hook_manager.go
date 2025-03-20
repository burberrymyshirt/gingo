package gingo

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

// HookType represents different hook points in the request lifecycle
type HookType int

const (
	BeforeRequest HookType = iota
	AfterRequest
	AfterPanic
)

// HookManager manages hooks for different lifecycle events
type HookManager struct {
	hooks          map[HookType][]HandlerFunc
	bootstrapHooks []gin.HandlerFunc
}

// NewHookManager creates a new hook manager
func NewHookManager() *HookManager {
	h := &HookManager{
		hooks:          make(map[HookType][]HandlerFunc),
		bootstrapHooks: getBootstrapFuncs(),
	}
	return h
}

func getBootstrapFuncs() []gin.HandlerFunc {
	var funcs []gin.HandlerFunc
	funcs = append(funcs, func(ginContext *gin.Context) {
		gingoContext := &Context{Context: ginContext}
		ginContext.Set("gingo_context", gingoContext)
		ginContext.Next()
	})

	return funcs
}

// AddHook adds a hook for a specific lifecycle event
func (m *HookManager) AddHook(hookType HookType, hook HandlerFunc) {
	m.hooks[hookType] = append(m.hooks[hookType], hook)
}

// executeHooks runs all hooks for a specific hook type
func (m *HookManager) executeHooks(
	hookType HookType,
	ginContext *gin.Context,
	additionalHooks ...HandlerFunc,
) {
	// Runs predefined global hooks, along with additional request middleware
	for _, hook := range append(m.hooks[hookType], additionalHooks...) {
		h := hook // copy to avoid closure issues
		context, ok := ginContext.Get("gingo_context")
		if !ok {
			panic("gingo context does not exist")
		}
		c := context.(*Context)
		h(c)
	}
}

func (m *HookManager) mapHandlers(routeDef *RouteDefinition) []gin.HandlerFunc {
	var ginHandlers []gin.HandlerFunc

	// Add bootstrapHooks
	ginHandlers = append(ginHandlers, m.bootstrapHooks...)

	// Add a middleware for "before request" hooks
	if len(append(m.hooks[BeforeRequest], routeDef.PreRequestMiddleware...)) > 0 {
		ginHandlers = append(ginHandlers, func(ginContext *gin.Context) {
			m.executeHooks(BeforeRequest, ginContext, routeDef.PreRequestMiddleware...)
			ginContext.Next()
		})
	}

	// Convert gingo handler to gin.HandlerFunc
	handler := routeDef.RequestHandler
	if handler == nil {
		panic(fmt.Sprintf("no request handler defined for %s", routeDef.Path))
	}
	ginHandlers = append(
		ginHandlers,
		func(ginContext *gin.Context) {
			context, ok := ginContext.Get("gingo_context")
			if !ok {
				panic("gingo context does not exist")
			}
			c := context.(*Context)
			h := handler
			h(c)
			c.Next() // call next, as normal request handlers don't do that
		},
	)

	// Add a middleware for "after request" hooks
	if len(append(m.hooks[AfterRequest], routeDef.PostRequestMiddleware...)) > 0 {
		ginHandlers = append(ginHandlers, func(ginContext *gin.Context) {
			m.executeHooks(AfterRequest, ginContext, routeDef.PostRequestMiddleware...)
		})
	}

	return ginHandlers
}
