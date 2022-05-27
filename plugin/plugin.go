package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const plugin_name = "krakend-debugger"

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

// ClientRegisterer is the symbol the plugin loader will try to load. It must implement the RegisterClient interface
var ClientRegisterer = registerer(plugin_name)

func (r registerer) RegisterClients(f func(name string, handler func(context.Context, map[string]interface{}) (http.Handler, error))) {
	f(string(r), r.registerClients)
}

func (r registerer) registerClients(ctx context.Context, extra map[string]interface{}) (h http.Handler, e error) {
	// check the passed configuration and initialize the plugin
	name, ok := extra["name"].(string)
	if !ok {
		return nil, errors.New("wrong config")
	}
	if name != string(r) {
		return nil, fmt.Errorf("unknown register %s", name)
	}
	// return the actual handler wrapping or your custom logic so it can be used as a replacement for the default http client
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		makeOriginalRequest(w, req)
	}), nil
}

// HandlerRegisterer is the symbol the plugin loader will try to load. It must implement the Registerer interface
var HandlerRegisterer = registerer(plugin_name)

func (r registerer) RegisterHandlers(f func(name string, handler func(context.Context, map[string]interface{}, http.Handler) (http.Handler, error))) {
	f(string(r), r.registerHandlers)
}

func (r registerer) registerHandlers(ctx context.Context, extra map[string]interface{}, old http.Handler) (h http.Handler, e error) {
	// check the passed configuration and initialize the plugin
	name, ok := extra["name"].(string)
	if !ok {
		return nil, errors.New("wrong config")
	}
	if name != string(r) {
		return nil, fmt.Errorf("unknown register %s", name)
	}

	// return the actual handler wrapping or your custom logic so it can be used as a replacement for the default http handler
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Create a unique request id - "github.com/google/uuid.NewString()" causes "plugin was built with a different version of package"
		t := time.Now().UnixNano()
		reqid := strconv.FormatInt(t, 36) + "_" + strconv.Itoa(rand.Intn(32768))
		req.Header["Requuid"] = []string{reqid}

		dump, err := httputil.DumpRequest(req, true)
		if err != nil {
			logF(err)
		}
		rqstr := string(dump)
		logD("req", rqstr)

		if true {
			// Request id and URL are the main keys
			keys := map[string]string{"id": reqid, "url": req.URL.Path}

			if false {
				// Add first 6 path segments, i.e. URL parts delimited by '/' as additional keys
				// E.g. URL /HardwarePricesDataManagement/v2/HardwarePrices/HardwarePrices
				// will fill seg0 = "", // Usually empty
				//           seg1 = "HardwarePricesDataManagement",
				//           seg2 ="v2",
				//           seg3 = "HardwarePrices",
				//           seg4 = "HardwarePrices"
				km := []string{"seg0", "seg1", "seg2", "seg3", "seg4", "seg5", "segremain"}
				segs := strings.SplitN(req.URL.Path, "/", len(km))
				for i, s := range segs {
					keys[km[i]] = s
				}
			}

			// Get non-blocking write client
			p := influx.NewPoint("payload",
				keys,
				map[string]interface{}{"request": rqstr},
				time.Now())
			writeAPI.WritePoint(p)
		}

		old.ServeHTTP(w, req)
	}), nil
}

// ModifierRegisterer is the symbol the plugin loader will be looking for. It must
// implement the plugin.Registerer interface
// https://github.com/luraproject/lura/blob/master/proxy/plugin/modifier.go#L71
var ModifierRegisterer = registerer(plugin_name)

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

func makeOriginalRequest(w http.ResponseWriter, req *http.Request) {
	// Depending on the HTTP client software, HTTP protocol version, and
	// any intermediaries between the client and the Go server, it may not
	// be possible to read from the Request.Body after writing to the
	// ResponseWriter. Cautious handlers should read the Request.Body
	// first, and then reply.

	// Dump incoming request
	dumprq, rqerr := httputil.DumpRequest(req, true)

	client := &http.Client{}
	// Send an HTTP request and returns an HTTP response object.
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		// return
	}
	defer resp.Body.Close()

	// headers
	for name, values := range resp.Header {
		w.Header()[name] = values
	}

	// status (must come after setting headers and before copying body)
	w.WriteHeader(resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	w.Write(body)

	if rqerr != nil {
		logF("rqerr", rqerr.Error())
	}
	rqstr := string(dumprq)
	logD("backend req", rqstr)

	dumprs, rserr := httputil.DumpResponse(resp, false)
	if rserr != nil {
		logF("rserr", rserr.Error())
	}
	var reqid string
	reqid = req.Header["Requuid"][0]
	res := string(dumprs) + string(body)
	logD("req_id ", reqid)
	logD("res ", res, "\n")

	if true {
		// get non-blocking write client
		p := influx.NewPoint("payload",
			map[string]string{"id": reqid},
			map[string]interface{}{"backend_request": rqstr, "response": res},
			time.Now())
		writeAPI.WritePoint(p)
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
