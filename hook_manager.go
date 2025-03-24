package gingo

// HookType represents different hook points in the request lifecycle
type HookType int

const (
	BeforeRequest HookType = iota
	AfterRequest
	AfterPanic
)

// HookManager manages hooks for different lifecycle events
type HookManager struct {
	hooks map[HookType][]HandlerFunc
}

// NewHookManager creates a new hook manager
func NewHookManager() *HookManager {
	h := &HookManager{
		hooks: make(map[HookType][]HandlerFunc),
	}

	return h
}

// RegisterHook adds a hook for a specific lifecycle event
func (m *HookManager) RegisterHook(hookType HookType, hook HandlerFunc) {
	m.hooks[hookType] = append(m.hooks[hookType], hook)
}
