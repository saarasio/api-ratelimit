package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
)

type Method int

const (
	GET Method = iota
	POST
	DELETE
	PATCH
)

type HttpMethod struct {
	method Method
	arg    interface{}
}

type Url string

type PostProxyArg struct {
	Name string `json:"Name"`
}

type PostServiceArg struct {
	Service_Name string `json: Service_Name`
	Fqdn         string `json: Fqdn`
}

type PostRouteArg struct {
	Route_Name   string `json: Route_Name`
	Route_Prefix string `json: Route_Prefix`
}

type PostUpstreamArg struct {
	Upstream_name    string `json: Upstream_name`
	Upstream_ip      string `json: Upstream_ip`
	Upstream_port    string `json: Upstream_port`
	Upstream_hc_path string `json: Upstream_hc_path`
	Upstream_weight  string `json: Upstream_weight`
}

type PostFilterArg struct {
	Filter_name   string `json: Filter_name `
	Filter_type   string `json: Filter_type`
	Filter_config string `json: Filter_config`
}

type PostGlobalConfigArg struct {
	Globalconfig_name string `json: Globalconfig_name `
	Globalconfig_type string `json: Globalconfig_type`
	Config            string `json: Config`
}

const base_url = `http://localhost:1323`

func pp(in []byte) string {
	var pj bytes.Buffer
	json.Indent(&pj, in, "", "    ")
	return string(pj.Bytes())
}

func err_out(res *http.Response, err error) {
	if err != nil {
		fmt.Println("Error\n")
	} else {
		b, _ := ioutil.ReadAll(res.Body)
		fmt.Println(pp(b))
	}
}

func err_check(err error, mesg string) {
	if err != nil {
		fmt.Println("Error " + mesg + " \n")
	}
}

// run http commands in sequence
func dohttp(cmds *map[int]map[HttpMethod]Url) {
	var keys []int
	for k := range *cmds {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {
		c := (*cmds)[k]
		for httpmethod, url := range c {
			switch httpmethod.method {
			case GET:
				res, err := http.Get(string(url))
				err_out(res, err)

			case POST:
				post_arg := httpmethod.arg
				post_arg_json, _ := json.Marshal(post_arg)
				res, err := http.Post(string(url), "application/json", bytes.NewBuffer(post_arg_json))
				err_out(res, err)

			case DELETE:
				req, err := http.NewRequest("DELETE", string(url), nil)
				err_check(err, "Delete failed for "+string(url)+"\n")
				client := &http.Client{}
				res, err := client.Do(req)
				res.Body.Close()
				err_out(res, err)

			case PATCH:
			default:
				// not supported
			}

		}
	}
}

//curl -s -X POST "http://localhost:1323/service/l/route/r/upstream/u" | jq
//curl -s -X POST "http://localhost:1323/proxy/gw/service/l" | jq
//curl -s -X POST localhost:1323/service/l/filter/lua_filter_1 | jq
//curl -s -X POST localhost:1323/service/l/route/r/filter/route_rl_1 | jq
//curl -s -X POST localhost:1323/proxy/gw/globalconfig/gc1 | jq

func main() {
	post_proxy_arg := PostProxyArg{
		Name: "test_gw",
	}

	post_service_arg := PostServiceArg{
		Service_Name: "test_svc",
		Fqdn:         "localhost",
	}

	post_route_arg := PostRouteArg{
		Route_Name:   "test_route",
		Route_Prefix: "/test_prefix",
	}

	post_upstream_arg := PostUpstreamArg{
		Upstream_name:    "test_upstream",
		Upstream_ip:      "localhost",
		Upstream_port:    "8888",
		Upstream_hc_path: "/",
		Upstream_weight:  "100",
	}

	var filter_cfg_lua = `
		function envoy_on_request(request_handle)
		   request_handle:logInfo("Hello World request");
		end
		
		function envoy_on_response(response_handle)
		   response_handle:logInfo("Hello World response");
		end
	`

	post_lua_filter_arg := PostFilterArg{
		Filter_name:   "test_filter_lua",
		Filter_type:   "http_filter_lua",
		Filter_config: filter_cfg_lua,
	}

	var filter_cfg_rl = `
	{
	  "descriptors" :
	  [
	    {
	      "generic_key":
	      {
	        "descriptor_value":"default"
	      }
	    }
	  ]
	}
	`

	post_rl_filter_arg := PostFilterArg{
		Filter_name:   "test_filter_rl",
		Filter_type:   "route_filter_ratelimit",
		Filter_config: filter_cfg_rl,
	}

	var gc_cfg_rl = `
		{
		  "domain": "enroute",
		  "descriptors" :
		  [
		    {
		      "key" : "generic_key",
		      "value" : "default",
		      "rate_limit" :
		      {
		        "unit" : "second",
		        "requests_per_unit" : 10
		      }
		    }
		  ]
		}

	`

	post_gc_arg := PostGlobalConfigArg{
		Globalconfig_name: "test_gc",
		Globalconfig_type: "globalconfig_ratelimit",
		Config:            gc_cfg_rl,
	}

	var setup_enroute_standalone = map[int]map[HttpMethod]Url{
		// sequence, {method, arg}, url

		// Dump Proxy gw if present
		25: map[HttpMethod]Url{HttpMethod{GET, nil}: Url(base_url + "/proxy/dump/gw")},

		// Create Proxy
		50: map[HttpMethod]Url{HttpMethod{POST, &post_proxy_arg}: Url(base_url + "/proxy")},

		// Create Service
		75: map[HttpMethod]Url{HttpMethod{POST, &post_service_arg}: Url(base_url + "/service")},

		// Create route for service
		100: map[HttpMethod]Url{
			HttpMethod{POST, &post_route_arg}: Url(base_url + "/service/" + post_service_arg.Service_Name + "/route")},

		// Create upstream
		125: map[HttpMethod]Url{
			HttpMethod{POST, &post_upstream_arg}: Url(base_url + "/upstream")},

		// Create filter lua
		150: map[HttpMethod]Url{
			HttpMethod{POST, &post_lua_filter_arg}: Url(base_url + "/filter")},

		// Create filter rl
		175: map[HttpMethod]Url{
			HttpMethod{POST, &post_rl_filter_arg}: Url(base_url + "/filter")},

		// Create globalconfig
		200: map[HttpMethod]Url{
			HttpMethod{POST, &post_gc_arg}: Url(base_url + "/globalconfig")},
	}

	var delete_enroute_standalone = map[int]map[HttpMethod]Url{
		// sequence, {method, arg}, url

		// Delete globalconfig
		23: map[HttpMethod]Url{HttpMethod{DELETE, nil}: Url(base_url + "/globalconfig/" + post_gc_arg.Globalconfig_name)},

		// Delete filter
		24: map[HttpMethod]Url{HttpMethod{DELETE, nil}: Url(base_url + "/filter/" + post_rl_filter_arg.Filter_name)},
		25: map[HttpMethod]Url{HttpMethod{DELETE, nil}: Url(base_url + "/filter/" + post_lua_filter_arg.Filter_name)},

		// Delete upstream
		50: map[HttpMethod]Url{HttpMethod{DELETE, nil}: Url(base_url + "/upstream/" + post_upstream_arg.Upstream_name)},

		// Delete route
		75: map[HttpMethod]Url{HttpMethod{DELETE, nil}: Url(base_url + "/service/" + post_service_arg.Service_Name + "/route/" + post_route_arg.Route_Name)},

		// Delete service
		100: map[HttpMethod]Url{HttpMethod{DELETE, nil}: Url(base_url + "/service/" + post_service_arg.Service_Name)},

		// Delete service
		125: map[HttpMethod]Url{HttpMethod{DELETE, nil}: Url(base_url + "/proxy/" + post_proxy_arg.Name)},
	}

	var create bool = true

	if create == true {
		dohttp(&setup_enroute_standalone)
	} else {
		dohttp(&delete_enroute_standalone)
	}

}
