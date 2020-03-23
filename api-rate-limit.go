package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
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
	mesg   string
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

func debug(data []byte, err error) {
	if err == nil {
		fmt.Printf("%s\n\n", data)
	} else {
		log.Fatalf("%s\n\n", err)
	}
}

func doreq(req *http.Request, err error, rstr, url string, dbg bool) {
	if err == nil {
		client := &http.Client{}
		if dbg {
			debug(httputil.DumpRequestOut(req, true))
		}
		res, err := client.Do(req)
		body, err := ioutil.ReadAll(res.Body)
		fmt.Printf("%s\n", pp(body))
		res.Body.Close()
		err_out(res, err)
	} else {
		fmt.Printf("Request run error while running [%s] url - [%s]\n", rstr, url)
	}

}

// run http commands in sequence
func dohttp(cmds *map[int]map[HttpMethod]Url, dbg bool) {
	var keys []int
	for k := range *cmds {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {
		c := (*cmds)[k]
		for httpmethod, url := range c {
			fmt.Println(httpmethod.mesg)
			switch httpmethod.method {
			case GET:
				req, err := http.NewRequest("GET", string(url), nil)
				doreq(req, err, "GET", string(url), dbg)

			case POST:
				var req *http.Request
				var err error
				if httpmethod.arg != nil {
					post_arg := httpmethod.arg
					post_arg_json, _ := json.Marshal(post_arg)
					req, err = http.NewRequest("POST", string(url), bytes.NewBuffer(post_arg_json))
					if err == nil {
						req.Header.Add("Content-Type", "application/json")
					} else {
						fmt.Printf("Request creation error while running POST url - [%s]\n", url)
					}

				} else {
					req, err = http.NewRequest("POST", string(url), nil)
				}

				doreq(req, err, "POST", string(url), dbg)

			case DELETE:
				req, err := http.NewRequest("DELETE", string(url), nil)
				doreq(req, err, "DELETE", string(url), dbg)

			case PATCH:
			default:
				// not supported
			}

		}
	}
}

func removews(with_ws []byte) []byte {
	with_ws = bytes.Replace(with_ws, []byte{10}, []byte{}, -1)
	with_ws = bytes.Replace(with_ws, []byte{9}, []byte{}, -1)
	with_ws = bytes.Replace(with_ws, []byte{92}, []byte{}, -1)

	return with_ws
}

func doop(op string, dbg bool) {

	post_proxy_arg := PostProxyArg{
		Name: "gw",
	}

	post_service_arg := PostServiceArg{
		Service_Name: "test_svc",
		Fqdn:         "localhost",
	}

	post_route_arg := PostRouteArg{
		Route_Name:   "test_route",
		Route_Prefix: "/",
	}

	post_upstream_arg := PostUpstreamArg{
		Upstream_name:    "test_upstream",
		Upstream_ip:      "localhost",
		Upstream_port:    "9001",
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
		}`

	gc_cfg_rl_b := removews([]byte(gc_cfg_rl))

	post_gc_arg := PostGlobalConfigArg{
		Globalconfig_name: "test_gc",
		Globalconfig_type: "globalconfig_ratelimit",
		Config:            string(gc_cfg_rl_b),
	}

	var urls = map[string]string{
		// Create Proxy
		"CREATE_P": base_url + "/proxy",

		// Create Service
		"CREATE_SVC": base_url + "/service",

		// Create Route
		"CREATE_RT": base_url +
			"/service/" + post_service_arg.Service_Name +
			"/route",

		// Create Upstream
		"CREATE_U": base_url + "/upstream",

		// Create filter
		"CREATE_FIL": base_url + "/filter",

		// Create globalconfig
		"CREATE_GC": base_url + "/globalconfig",

		// Delete globalconfig
		"DEL_GC": base_url +
			"/globalconfig/" + post_gc_arg.Globalconfig_name,

		// Delete rate limit route filter
		"DEL_RT_FIL": base_url +
			"/filter/" + post_rl_filter_arg.Filter_name,

		// Delete lua service filter
		"DEL_SVC_FIL": base_url +
			"/filter/" + post_lua_filter_arg.Filter_name,

		// Delete upstream
		"DEL_SVC_U": base_url +
			"/upstream/" + post_upstream_arg.Upstream_name,

		// Delete route
		"DEL_RT": base_url +
			"/service/" + post_service_arg.Service_Name +
			"/route/" + post_route_arg.Route_Name,

		// Delete service
		"DEL_SVC": base_url +
			"/service/" + post_service_arg.Service_Name,

		// Delete proxy
		"DEL_P": base_url +
			"/proxy/" + post_proxy_arg.Name,

		// Associate/Disassociate globalconfig from proxy
		"PROXY_GC": base_url +
			"/proxy/" + post_proxy_arg.Name +
			"/globalconfig/" + post_gc_arg.Globalconfig_name,

		// Associate/Disassociate filter from route
		"SVC_RT_FIL": base_url +
			"/service/" + post_service_arg.Service_Name +
			"/route/" + post_route_arg.Route_Name +
			"/filter/" + post_rl_filter_arg.Filter_name,

		// Associate/Disassociate filter from service
		"SVC_FIL": base_url +
			"/service/" + post_service_arg.Service_Name +
			"/filter/" + post_lua_filter_arg.Filter_name,

		// Associate/Disassociate upstream from route
		"SVC_U": base_url +
			"/service/" + post_service_arg.Service_Name +
			"/route/" + post_route_arg.Route_Name +
			"/upstream/" + post_upstream_arg.Upstream_name,

		// Associate/Disassociate service from proxy
		"PROXY_SVC": base_url +
			"/proxy/" + post_proxy_arg.Name +
			"/service/" + post_service_arg.Service_Name,

		// Dump Proxy
		"DUMP_P": base_url +
			"/proxy/dump/" + post_proxy_arg.Name,
	}

	var steps *map[int]map[HttpMethod]Url

	// init with no-op
	steps = &(map[int]map[HttpMethod]Url{})

	switch op {
	case "create":
		var setup_enroute_standalone = map[int]map[HttpMethod]Url{
			// sequence, {method, arg}, url

			25:  map[HttpMethod]Url{HttpMethod{GET, nil, "-- GET PROXY --"}: Url(urls["DUMP_P"])},
			50:  map[HttpMethod]Url{HttpMethod{POST, &post_proxy_arg, "-- POST PROXY --"}: Url(urls["CREATE_P"])},
			75:  map[HttpMethod]Url{HttpMethod{POST, &post_service_arg, "-- POST SVC --"}: Url(urls["CREATE_SVC"])},
			100: map[HttpMethod]Url{HttpMethod{POST, &post_route_arg, "-- POST RT --"}: Url(urls["CREATE_RT"])},
			125: map[HttpMethod]Url{HttpMethod{POST, &post_upstream_arg, "-- POST U --"}: Url(urls["CREATE_U"])},
			130: map[HttpMethod]Url{HttpMethod{POST, nil, "-- POST SVC/R/U --"}: Url(urls["SVC_U"])},
			140: map[HttpMethod]Url{HttpMethod{POST, nil, "-- POST PROXY/SVC --"}: Url(urls["PROXY_SVC"])},
			150: map[HttpMethod]Url{HttpMethod{POST, &post_lua_filter_arg, "-- POST FIL --"}: Url(urls["CREATE_FIL"])},
			160: map[HttpMethod]Url{HttpMethod{POST, nil, "-- POST SVC/FIL --"}: Url(urls["SVC_FIL"])},
			175: map[HttpMethod]Url{HttpMethod{POST, &post_rl_filter_arg, "-- POST FIL --"}: Url(urls["CREATE_FIL"])},
			185: map[HttpMethod]Url{HttpMethod{POST, nil, "-- POST SVC/R/FIL --"}: Url(urls["SVC_RT_FIL"])},
			200: map[HttpMethod]Url{HttpMethod{POST, &post_gc_arg, "-- POST GC --"}: Url(urls["CREATE_GC"])},
			225: map[HttpMethod]Url{HttpMethod{POST, &post_gc_arg, "-- POST PROXY/GC --"}: Url(urls["PROXY_GC"])},
		}

		steps = &setup_enroute_standalone
	case "delete":
		var delete_enroute_standalone = map[int]map[HttpMethod]Url{
			// sequence, {method, arg}, url

			10:  map[HttpMethod]Url{HttpMethod{DELETE, nil, "-- DIS GC --"}: Url(urls["PROXY_GC"])},
			12:  map[HttpMethod]Url{HttpMethod{DELETE, nil, "-- DEL GC --"}: Url(urls["DEL_GC"])},
			20:  map[HttpMethod]Url{HttpMethod{DELETE, nil, "-- DIS RT FIL --"}: Url(urls["SVC_RT_FIL"])},
			22:  map[HttpMethod]Url{HttpMethod{DELETE, nil, "-- DEL FIL --"}: Url(urls["DEL_RT_FIL"])},
			24:  map[HttpMethod]Url{HttpMethod{DELETE, nil, "-- DIS SVC FIL --"}: Url(urls["SVC_FIL"])},
			25:  map[HttpMethod]Url{HttpMethod{DELETE, nil, "-- DEL FIL --"}: Url(urls["DEL_SVC_FIL"])},
			45:  map[HttpMethod]Url{HttpMethod{DELETE, nil, "-- DIS U --"}: Url(urls["SVC_U"])},
			50:  map[HttpMethod]Url{HttpMethod{DELETE, nil, "-- DEL U --"}: Url(urls["DEL_SVC_U"])},
			75:  map[HttpMethod]Url{HttpMethod{DELETE, nil, "-- DEL RT --"}: Url(urls["DEL_RT"])},
			80:  map[HttpMethod]Url{HttpMethod{DELETE, nil, "-- DIS PROXY/SVC --"}: Url(urls["PROXY_SVC"])},
			100: map[HttpMethod]Url{HttpMethod{DELETE, nil, "-- DEL SVC --"}: Url(urls["DEL_SVC"])},
			125: map[HttpMethod]Url{HttpMethod{DELETE, nil, "-- DEL PROXY --"}: Url(urls["DEL_P"])},
		}

		steps = &delete_enroute_standalone
	case "show":

		var show_enroute_standalone = map[int]map[HttpMethod]Url{
			25:  map[HttpMethod]Url{HttpMethod{GET, nil, " -- DUMP PROXY -- "}: Url(urls["DUMP_P"])},
			50:  map[HttpMethod]Url{HttpMethod{GET, nil, " -- GET PROXY -- "}: Url(urls["CREATE_P"])},
			75:  map[HttpMethod]Url{HttpMethod{GET, nil, " -- GET SVC -- "}: Url(urls["CREATE_SVC"])},
			100: map[HttpMethod]Url{HttpMethod{GET, nil, " -- GET RT -- "}: Url(urls["CREATE_RT"])},
			125: map[HttpMethod]Url{HttpMethod{GET, nil, " -- GET U -- "}: Url(urls["CREATE_U"])},
			150: map[HttpMethod]Url{HttpMethod{GET, nil, " -- GET FIL -- "}: Url(urls["CREATE_FIL"])},
			175: map[HttpMethod]Url{HttpMethod{GET, nil, " -- GET GC -- "}: Url(urls["CREATE_GC"])},
		}

		steps = &show_enroute_standalone

	default:
		fmt.Printf("Operation [%s] not supported\n", op)
	}

	dohttp(steps, dbg)
}

func main() {
	op := flag.String("op", "show", "[create | delete | show]")
	dbg := flag.Bool("dbg", false, "[true | false]")
	flag.Parse()
	doop(*op, *dbg)
}
