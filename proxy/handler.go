package proxy

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/naoina/toml"
)

type config struct {
	Upstream string
}

type Handler struct {
	Config config
	client *http.Client
}

func New() *Handler {
	h := &Handler{
		client: &http.Client{},
	}

	file, err := os.Open("config.toml")
	if err != nil {
		log.Fatal("unable to open config.toml", err)
	}

	decoder := toml.NewDecoder(file)
	err = decoder.Decode(&h.Config)
	if err != nil {
		log.Fatal("unable to decode config.toml", err)
	}
	return h
}

// encodeURI takes a case sensitive encoded request method
// and returns a underscore separated version of the string
func (h *Handler) encodeURI(method string) string {
	r := regexp.MustCompile("(_([a-z]+)|([A-Z][a-z]+))")
	rseg := r.FindAllString(method, -1)

	// re-construct upstream REST based URI
	var useg []string
	for _, m := range rseg {
		useg = append(useg, strings.Trim(m, "_"))
	}
	upath := strings.Join(useg, "_")
	upath = strings.ToLower(upath)
	upath = h.Config.Upstream + upath
	return upath
}

// callUpstream calls a HTTP uri and returns http.Response
func (h *Handler) callUpstream(method, uri string) (*http.Response, error) {
	req, err := http.NewRequest(method, uri, nil)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.Do(req)
	return resp, nil
}

// TranslateRequest takes a JSON-RPC request and routes it to a REST API upstream
func (h *Handler) TranslateRequest(c *gin.Context) {
	var request struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		ID      json.RawMessage `json:"id"`
		Params  json.RawMessage `json:"params"`
	}

	// decode incoming JSON-RPC request
	if c.Bind(&request) != nil {
		return
	}

	// we default to upstream GET requests
	umethod := "GET"

	// `method` will be in format of "addonnamespace_restApiPath"
	// we will transform the restApiPath format to rest_api_path
	upath := h.encodeURI(request.Method)

	// parse request `params` for optional arguments
	var requestParams = []map[string]string{}
	json.Unmarshal([]byte(request.Params), &requestParams)
	if len(requestParams) == 1 {
		// path argument to add to upstream URI
		upathParam, ok := requestParams[0]["path"]
		if ok {
			upath = upath + "/" + upathParam
		}
		// upstream request param
		umethodParam, ok := requestParams[0]["method"]
		if ok {
			umethod = umethodParam
		}
		// take all other params as optional URL arguments
		uvalues := make(url.Values)
		for k, v := range requestParams[0] {
			if k == "path" || k == "method" {
				continue
			}
			uvalues.Add(k, v)
		}
		upathvalues := uvalues.Encode()
		if upathvalues != "" {
			upath += "?" + upathvalues
		}
	}

	// construct upstream HTTP request
	// default to GET request method unless otherwise specified in params
	resp, err := h.callUpstream(umethod, upath)
	if err != nil {
		log.Fatal(err)
	}
	// read raw upstream response and return as is
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	c.String(resp.StatusCode, string(data))
}
