package jsonrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/autobrr/autobrr/pkg/errors"
)

type Client interface {
	Call(method string, params ...interface{}) (*RPCResponse, error)
}

type RPCRequest struct {
	JsonRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

func NewRequest(method string, params ...interface{}) *RPCRequest {
	return &RPCRequest{
		JsonRPC: "2.0",
		Method:  method,
		Params:  Params(params...),
		ID:      0,
	}
}

type RPCResponse struct {
	JsonRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      int         `json:"id"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return strconv.Itoa(e.Code) + ":" + e.Message
}

type HTTPError struct {
	Code int
	err  error
}

func (e *HTTPError) Error() string {
	return e.err.Error()
}

type rpcClient struct {
	endpoint   string
	httpClient *http.Client
	headers    map[string]string
}

type ClientOpts struct {
	HTTPClient *http.Client
	Headers    map[string]string
}

type RPCResponses []*RPCResponse

func NewClient(endpoint string) Client {
	return NewClientWithOpts(endpoint, nil)
}

func NewClientWithOpts(endpoint string, opts *ClientOpts) Client {
	c := &rpcClient{
		endpoint:   endpoint,
		httpClient: &http.Client{},
		headers:    make(map[string]string),
	}

	if opts == nil {
		return c
	}

	if opts.HTTPClient != nil {
		c.httpClient = opts.HTTPClient
	}

	if opts.Headers != nil {
		for k, v := range opts.Headers {
			c.headers[k] = v
		}
	}

	return c
}

func (c *rpcClient) Call(method string, params ...interface{}) (*RPCResponse, error) {
	request := RPCRequest{
		ID:      1,
		JsonRPC: "2.0",
		Method:  method,
		Params:  Params(params...),
	}

	return c.doCall(request)
}

func (c *rpcClient) newRequest(req interface{}) (*http.Request, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal request")
	}

	request, err := http.NewRequest("POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "error creating request")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	for k, v := range c.headers {
		request.Header.Set(k, v)
	}

	return request, nil
}

func (c *rpcClient) doCall(request RPCRequest) (*RPCResponse, error) {

	httpRequest, err := c.newRequest(request)
	if err != nil {
		return nil, errors.Wrap(err, "could not create rpc http request")
	}

	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return nil, errors.Wrap(err, "error during rpc http request")
	}

	defer httpResponse.Body.Close()

	var rpcResponse *RPCResponse
	decoder := json.NewDecoder(httpResponse.Body)
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	err = decoder.Decode(&rpcResponse)

	if err != nil {
		if httpResponse.StatusCode >= 400 {
			return nil, errors.Wrap(err, fmt.Sprintf("rpc call %v() on %v status code: %v. Could not decode body to rpc response", request.Method, httpRequest.URL.String(), httpResponse.StatusCode))
		}
		//	if res.StatusCode == http.StatusUnauthorized {
		//		return nil, errors.New("unauthorized: bad credentials")
		//	} else if res.StatusCode == http.StatusForbidden {
		//		return nil, nil
		//	} else if res.StatusCode == http.StatusTooManyRequests {
		//		return nil, nil
		//	} else if res.StatusCode == http.StatusBadRequest {
		//		return nil, nil
		//	} else if res.StatusCode == http.StatusNotFound {
		//		return nil, nil
		//	} else if res.StatusCode == http.StatusServiceUnavailable {
		//		return nil, nil
		//	}
	}

	if rpcResponse == nil {
		return nil, errors.New("rpc call %v() on %v status code: %v. rpc response missing", request.Method, httpRequest.URL.String(), httpResponse.StatusCode)
	}

	return rpcResponse, nil
}

func Params(params ...interface{}) interface{} {
	var finalParams interface{}

	// if params was nil skip this and p stays nil
	if params != nil {
		switch len(params) {
		case 0: // no parameters were provided, do nothing so finalParam is nil and will be omitted
		case 1: // one param was provided, use it directly as is, or wrap primitive types in array
			if params[0] != nil {
				var typeOf reflect.Type

				// traverse until nil or not a pointer type
				for typeOf = reflect.TypeOf(params[0]); typeOf != nil && typeOf.Kind() == reflect.Ptr; typeOf = typeOf.Elem() {
				}

				if typeOf != nil {
					// now check if we can directly marshal the type or if it must be wrapped in an array
					switch typeOf.Kind() {
					// for these types we just do nothing, since value of p is already unwrapped from the array params
					case reflect.Struct:
						finalParams = params[0]
					case reflect.Array:
						finalParams = params[0]
					case reflect.Slice:
						finalParams = params[0]
					case reflect.Interface:
						finalParams = params[0]
					case reflect.Map:
						finalParams = params[0]
					default: // everything else must stay in an array (int, string, etc)
						finalParams = params
					}
				}
			} else {
				finalParams = params
			}
		default: // if more than one parameter was provided it should be treated as an array
			finalParams = params
		}
	}

	return finalParams
}

func (r *RPCResponse) GetObject(toType interface{}) error {
	js, err := json.Marshal(r.Result)
	if err != nil {
		return errors.Wrap(err, "could not marshal object")
	}

	err = json.Unmarshal(js, toType)
	if err != nil {
		return errors.Wrap(err, "could not unmarshal object")
	}

	return nil
}
