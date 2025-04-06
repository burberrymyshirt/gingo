package gingo

import (
	"errors"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator"
)

type Context struct {
	*gin.Context
	engine   *Engine
	routeDef *RouteDefinition
}

type CachedFieldInfo struct {
	JSONFieldMap   map[string]string
	RequiredFields []string
	LastAccessedAt time.Time
}

// Cache with expiration management
var (
	fieldInfoCache  = sync.Map{}
	cacheMutex      = &sync.Mutex{}
	cacheExpiration = 1 * time.Hour // Cache entries expire after 1 hour of non-use
)

func InitializeCacneCleanup() {
	go func() {
		t := time.NewTicker(time.Minute * 5)
		defer t.Stop()

		for range t.C {
			cleanupContextCache()
		}
	}()
}

func cleanupContextCache() {
	now := time.Now()
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	fieldInfoCache.Range(
		func(key, value interface{}) bool {
			info, ok := value.(*CachedFieldInfo)
			if !ok {
				fieldInfoCache.Delete(key)
				return true
			}

			if now.Sub(info.LastAccessedAt) > cacheExpiration {
				fieldInfoCache.Delete(key)
			}
			return true
		},
	)
}

// HandlerName returns the main handler's name. For example if the handler is "handleGetUsers()",
// this function will return "main.handleGetUsers".
func (c *Context) HandlerName() string {
	return runtime.FuncForPC(reflect.ValueOf(c.routeDef.RequestHandler).Pointer()).Name()
}

// HandlerNames returns a list of all registered handlers for this context in descending order,
// following the semantics of HandlerName()
func (c *Context) HandlerNames() []string {
	hn := make([]string, 0, len(c.routeDef.getHandlers()))
	for _, val := range c.routeDef.getHandlers() {
		if val == nil {
			continue
		}
		hn = append(hn, runtime.FuncForPC(reflect.ValueOf(val).Pointer()).Name())
	}
	return hn
}

// Handler returns the main handler.
func (c *Context) Handler() HandlerFunc {
	return c.routeDef.RequestHandler
}

// Reset prepares the context for reuse
func (c *Context) Reset() {
	c.routeDef = nil
	// Reset other custom fields
}

// Next is a convenience method to call Next on the underlying gin.Context
func (c *Context) Next() {
	c.Context.Next()
}

//func (c *Context) ContentType() string {
//	return c.Context.ContentType()
//}
//
//func (c *Context) Request() *http.Request {
//	return c.Context.Request
//}

// ShouldBind checks the Method and Content-Type to select a binding engine automatically,
// Depending on the "Content-Type" header different bindings are used, for example:
//
//	"application/json" --> JSON binding
//	"application/xml"  --> XML binding
//
// It parses the request's body as JSON if Content-Type == "application/json" using JSON or XML as a JSON input.
// It decodes the json payload into the struct specified as a pointer.
// Like c.Bind() but this method does not set the response status code to 400 or abort if input is not valid.
func (c *Context) ShouldBind(obj interface{}) error {
	b := binding.Default(c.Context.Request.Method, c.Context.ContentType())
	return c.ShouldBindWith(obj, b)
}

// ShouldBindJSON is a shortcut for c.ShouldBindWith(obj, binding.JSON).
func (c *Context) ShouldBindJSON(obj interface{}) error {
	return c.ShouldBindWith(obj, binding.JSON)
}

// ShouldBindXML is a shortcut for c.ShouldBindWith(obj, binding.XML).
func (c *Context) ShouldBindXML(obj interface{}) error {
	return c.ShouldBindWith(obj, binding.XML)
}

// ShouldBindQuery is a shortcut for c.ShouldBindWith(obj, binding.Query).
func (c *Context) ShouldBindQuery(obj interface{}) error {
	return c.ShouldBindWith(obj, binding.Query)
}

// ShouldBindYAML is a shortcut for c.ShouldBindWith(obj, binding.YAML).
func (c *Context) ShouldBindYAML(obj interface{}) error {
	return c.ShouldBindWith(obj, binding.YAML)
}

// ShouldBindTOML is a shortcut for c.ShouldBindWith(obj, binding.TOML).
func (c *Context) ShouldBindTOML(obj interface{}) error {
	return c.ShouldBindWith(obj, binding.TOML)
}

// ShouldBindPlain is a shortcut for c.ShouldBindWith(obj, binding.Plain).
func (c *Context) ShouldBindPlain(obj interface{}) error {
	return c.ShouldBindWith(obj, binding.Plain)
}

// ShouldBindHeader is a shortcut for c.ShouldBindWith(obj, binding.Header).
func (c *Context) ShouldBindHeader(obj interface{}) error {
	return c.ShouldBindWith(obj, binding.Header)
}

// ShouldBindUri binds the passed struct pointer using the specified binding engine.
func (c *Context) ShouldBindUri(obj interface{}) error {
	err := c.Context.ShouldBindUri(obj)
	if err != nil {
		return c.parseError(err, obj)
	}
	return nil
}

// ShouldBindWith binds the passed struct pointer using the specified binding engine.
// See the binding package.
func (c *Context) ShouldBindWith(obj interface{}, b binding.Binding) error {
	err := b.Bind(c.Request, obj)
	if err != nil {
		return c.parseError(err, obj)
	}
	return nil
}

// ShouldBindBodyWith is similar with ShouldBindWith, but it stores the request
// body into the context, and reuse when it is called again.
//
// NOTE: This method reads the body before binding. So you should use
// ShouldBindWith for better performance if you need to call only once.
func (c *Context) ShouldBindBodyWith(obj interface{}, bb binding.BindingBody) (err error) {
	err = c.Context.ShouldBindBodyWith(obj, bb)
	if err != nil {
		return c.parseError(err, obj)
	}

	return nil
}

// ShouldBindBodyWithJSON is a shortcut for c.ShouldBindBodyWith(obj, binding.JSON).
func (c *Context) ShouldBindBodyWithJSON(obj interface{}) error {
	return c.ShouldBindBodyWith(obj, binding.JSON)
}

// ShouldBindBodyWithXML is a shortcut for c.ShouldBindBodyWith(obj, binding.XML).
func (c *Context) ShouldBindBodyWithXML(obj interface{}) error {
	return c.ShouldBindBodyWith(obj, binding.XML)
}

// ShouldBindBodyWithYAML is a shortcut for c.ShouldBindBodyWith(obj, binding.YAML).
func (c *Context) ShouldBindBodyWithYAML(obj interface{}) error {
	return c.ShouldBindBodyWith(obj, binding.YAML)
}

// ShouldBindBodyWithTOML is a shortcut for c.ShouldBindBodyWith(obj, binding.TOML).
func (c *Context) ShouldBindBodyWithTOML(obj interface{}) error {
	return c.ShouldBindBodyWith(obj, binding.TOML)
}

// ShouldBindBodyWithPlain is a shortcut for c.ShouldBindBodyWith(obj, binding.Plain).
func (c *Context) ShouldBindBodyWithPlain(obj interface{}) error {
	return c.ShouldBindBodyWith(obj, binding.Plain)
}

func (c *Context) parseError(err error, obj interface{}) error {
	if err == nil {
		return nil
	}

	objType := reflect.TypeOf(obj)
	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}
	typeName := objType.PkgPath() + "." + objType.Name()

	// Get or create cache entry
	fieldInfo, ok := getFieldInfo(typeName, obj)
	if !ok {
		return errors.New("failed to process validation error")
	}

	// Handle EOF error (empty request body)
	if err.Error() == "EOF" {
		if len(fieldInfo.RequiredFields) == 0 {
			return nil
		}
		return errors.New("request body cannot be empty, required fields: " +
			strings.Join(fieldInfo.RequiredFields, ", "))
	}

	// Process validation errors
	var validatorErrors validator.ValidationErrors
	if !errors.As(err, &validatorErrors) {
		return errors.New("invalid request: " + err.Error())
	}

	// Format the validation errors
	errorMsgs := make([]string, len(validatorErrors))
	for i, fieldError := range validatorErrors {
		fieldName := fieldError.Field()
		if jsonName, exists := fieldInfo.JSONFieldMap[fieldName]; exists {
			fieldName = jsonName
		}
		errorMsgs[i] = fieldName + ": " + fieldError.Tag()
	}

	return errors.New("invalid request: " + strings.Join(errorMsgs, ", "))
}

// Get field information from cache or create new entry
func getFieldInfo(typeName string, obj interface{}) (*CachedFieldInfo, bool) {
	// Try to get from cache first
	if cached, found := fieldInfoCache.Load(typeName); found {
		info := cached.(*CachedFieldInfo)
		cacheMutex.Lock()
		info.LastAccessedAt = time.Now()
		cacheMutex.Unlock()
		return info, true
	}

	// Not found in cache, create new
	info := analyzeType(obj)

	// Store in cache
	cacheMutex.Lock()
	fieldInfoCache.Store(typeName, info)
	cacheMutex.Unlock()

	return info, true
}

func analyzeType(obj interface{}) *CachedFieldInfo {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()

	info := &CachedFieldInfo{
		JSONFieldMap:   make(map[string]string),
		RequiredFields: []string{},
		LastAccessedAt: time.Now(),
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" || len(jsonTag) == 0 {
			continue
		}

		jsonFieldName := strings.Split(jsonTag, ",")[0]
		info.JSONFieldMap[field.Name] = jsonFieldName

		validateTag := field.Tag.Get("binding")
		if strings.Contains(validateTag, "required") {
			info.RequiredFields = append(info.RequiredFields, jsonFieldName)
		}
	}

	return info
}
