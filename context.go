package gingo

import (
	"errors"
	"reflect"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator"
)

type Context struct {
	*gin.Context
	engine   *Engine
	routeDef *RouteDefinition
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

	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()

	if err.Error() == "EOF" {
		// Reflect on the obj to find required fields
		var requiredFields []string
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" || len(jsonTag) <= 0 {
				continue
			}

			jsonFieldName := strings.Split(jsonTag, ",")[0]
			validateTag := field.Tag.Get("binding")
			if !strings.Contains(validateTag, "required") {
				continue
			}
			requiredFields = append(requiredFields, jsonFieldName)
		}

		if len(requiredFields) <= 0 {
			return nil
		}

		return errors.New(
			"request body cannot be empty, required fields: " + strings.Join(
				requiredFields,
				", ",
			),
		)
	}

	var validatorErrors validator.ValidationErrors
	ok := errors.As(err, &validatorErrors)
	if !ok {
		return errors.New("invalid request: " + err.Error())
	}
	// Map to hold the json field names
	jsonTagMap := make(map[string]string)
	// Reflect the obj to find the json tag names
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" || len(jsonTag) <= 0 {
			continue
		}
		jsonFieldName := strings.Split(jsonTag, ",")[0]
		jsonTagMap[field.Name] = jsonFieldName
	}

	out := make([]string, len(validatorErrors))
	for i, fieldError := range validatorErrors {
		fieldName := fieldError.Field()
		if jsonName, exists := jsonTagMap[fieldName]; exists {
			fieldName = jsonName
		}
		out[i] = fieldName + ": " + fieldError.Tag()
	}
	return errors.New("invalid request: " + strings.Join(out, ", "))
}
