package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
)

var (
	logger Logger = nil
	debug  int    = 1
)

// RequestWrapper is an interface for passing proxy request between the krakend pipe
// and the loaded plugins
type RequestWrapper interface {
	Params() map[string]string
	Headers() map[string][]string
	Body() io.ReadCloser
	Method() string
	URL() *url.URL
	Query() url.Values
	Path() string
}

func modifier(req RequestWrapper) requestWrapper {
	return requestWrapper{
		params:  req.Params(),
		headers: req.Headers(),
		body:    req.Body(),
		method:  req.Method(),
		url:     req.URL(),
		query:   req.Query(),
		path:    req.Path(),
	}
}

// ResponseWrapper is an interface for passing proxy response between the krakend pipe
// and the loaded plugins
type ResponseWrapper interface {
	Data() map[string]interface{}
	Io() io.Reader
	IsComplete() bool
	StatusCode() int
	Headers() map[string][]string
}

// Logger is an interface for logging functions
type Logger interface {
	Debug(v ...interface{})
	Info(v ...interface{})
	Warning(v ...interface{})
	Error(v ...interface{})
	Critical(v ...interface{})
	Fatal(v ...interface{})
}

type requestWrapper struct {
	method  string
	url     *url.URL
	query   url.Values
	path    string
	body    io.ReadCloser
	params  map[string]string
	headers map[string][]string
}

func (r requestWrapper) Method() string               { return r.method }
func (r requestWrapper) URL() *url.URL                { return r.url }
func (r requestWrapper) Query() url.Values            { return r.query }
func (r requestWrapper) Path() string                 { return r.path }
func (r requestWrapper) Body() io.ReadCloser          { return r.body }
func (r requestWrapper) Params() map[string]string    { return r.params }
func (r requestWrapper) Headers() map[string][]string { return r.headers }

type registerer string

// ModifierRegisterer is the symbol the plugin loader will be looking for. It must
// implement the plugin.Registerer interface
// https://github.com/luraproject/lura/blob/master/proxy/plugin/modifier.go#L71
var ModifierRegisterer = registerer("krakend-debugger")

// RegisterModifiers is the function the plugin loader will call to register the
// modifier(s) contained in the plugin using the function passed as argument.
// f will register the factoryFunc under the name and mark it as a request
// and/or response modifier.
func (r registerer) RegisterModifiers(f func(
	name string,
	factoryFunc func(map[string]interface{}) func(interface{}) (interface{}, error),
	appliesToRequest bool,
	appliesToResponse bool,
)) {
	f(string(r)+"-request", r.requestModifierFactory, true, false)
	f(string(r)+"-response", r.responseModifierFactory, false, true)
	logD(fmt.Sprintf("[PLUGIN: %s] Plugin Registered!", string(r)))
}

// RegisterLogger is the function the plugin loader will call to register the
// logger to be used in the plugin by passing it as an argument to the function.
func (registerer) RegisterLogger(in interface{}) {
	l, ok := in.(Logger)
	if !ok {
		fmt.Println(fmt.Sprintf("[PLUGIN: %s] No logger registered!", ModifierRegisterer))
		return
	}

	logger = l
	logD(fmt.Sprintf("[PLUGIN: %s] Logger loaded", ModifierRegisterer))
}

var errUnknownType = errors.New("unknow request type")

func (r registerer) requestModifierFactory(
	cfg map[string]interface{},
) func(interface{}) (interface{}, error) {
	// return the modifier
	logD(fmt.Sprintf("[PLUGIN: %s] Request Modifier injected", string(r)))

	return func(input interface{}) (interface{}, error) {
		logD(fmt.Sprintf("[PLUGIN: %s] RequestModifier got called with: %s", ModifierRegisterer, input))

		req, ok := input.(RequestWrapper)
		if !ok {
			return nil, errUnknownType
		}

		r := modifier(req)

		// Get request body
		body, err := ioutil.ReadAll(r.body)
		if err != nil {
			logD(fmt.Sprintf("[PLUGIN: %s] Can't read request body %v", ModifierRegisterer, err))
		}

		logD(fmt.Sprintf("[PLUGIN: %s] params: %v", ModifierRegisterer, r.params))
		logD(fmt.Sprintf("[PLUGIN: %s] headers: %v", ModifierRegisterer, r.params))
		logD(fmt.Sprintf("[PLUGIN: %s] method: %v", ModifierRegisterer, r.method))
		logD(fmt.Sprintf("[PLUGIN: %s] url: %v", ModifierRegisterer, r.url))
		logD(fmt.Sprintf("[PLUGIN: %s] query: %v", ModifierRegisterer, r.query))
		logD(fmt.Sprintf("[PLUGIN: %s] path: %v", ModifierRegisterer, r.path))
		logD(fmt.Sprintf("[PLUGIN: %s] body: %v", ModifierRegisterer, body))

		return r, nil
	}
}

func (r registerer) responseModifierFactory(
	cfg map[string]interface{},
) func(interface{}) (interface{}, error) {
	logD(fmt.Sprintf("[PLUGIN: %s] Response Modifier injected", ModifierRegisterer))

	// return the modifier
	return func(input interface{}) (interface{}, error) {
		logD(fmt.Sprintf("[PLUGIN: %s] ResponseModifier got called with: %s", ModifierRegisterer, input))

		resp, ok := input.(ResponseWrapper)
		if !ok {
			return nil, errUnknownType
		}

		logD(fmt.Sprintf("[PLUGIN: %s] data: %v", ModifierRegisterer, resp.Data()))
		logD(fmt.Sprintf("[PLUGIN: %s] is complete: %v", ModifierRegisterer, resp.IsComplete()))
		logD(fmt.Sprintf("[PLUGIN: %s] headers: %v", ModifierRegisterer, resp.Headers()))
		logD(fmt.Sprintf("[PLUGIN: %s] status code: %v", ModifierRegisterer, resp.StatusCode()))

		return input, nil
	}
}

func logD(v ...interface{}) {
	if logger != nil {
		logger.Debug(v...)
	} else if debug > 0 {
		fmt.Print("DEBUG: ")
		fmt.Println(v...)
	}
}

func logF(v ...interface{}) {
	if logger != nil {
		logger.Fatal(v...)
	} else {
		fmt.Print("FATAL: ")
		fmt.Println(v...)
	}
}

func init() {
	logD(fmt.Sprintf("[PLUGIN: %s] Initialized", ModifierRegisterer))
}

func main() {}
