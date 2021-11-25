package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/jefflinse/melatonin/json"
	"github.com/jefflinse/melatonin/mt"
)

// This example shows how to use melatonin with all of its configurable settings.
func FullExample() {
	// Create a mock server for this example. See startFullExampleServer() below.
	server := startFullExampleServer()
	defer server.Close()

	// Testing a Base URL
	//
	// myURL is a test context that can be used to create test cases that
	// target a specific base URL. This is useful for testing actual running
	// services that are running locally or remotely.
	//
	// The HTTP client can be configured however necessary to ensure
	// compatibility with the service being tested.
	myURL := mt.NewURLContext(server.URL).WithHTTPClient(http.DefaultClient)

	// Testing an HTTP Handler
	//
	// myHandler is a test context that can be used to create test cases that
	// target a specific handler. This is useful for testing HTTP handler logic
	// directly, making tests created using this context suitable for unit tests.

	// Anything satifying the http.Handler interface can be tested as a handler.
	mux := http.NewServeMux()
	mux.HandleFunc("/foo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("Hello, world!"))
	})

	myHandler := mt.NewHandlerContext(mux)

	// A custom HTTP request can be created to provide complete control over
	// all aspects of a particular test case, if needed. See usage of customReq
	// below.
	customReq, _ := http.NewRequest("GET", server.URL+"/foo", nil)

	// Creating a custom test runner provides the ability to customize the
	// behavior of the test runner.
	//
	// To run tests using the default runner, use
	//
	//     mt.RunTests()
	//
	runner := mt.NewTestRunner().WithContinueOnFailure(true).WithRequestTimeout(1 * time.Second)
	results, err := runner.RunTests([]mt.TestCase{

		myURL.GET("/foo", "Fetch foo with a custom timeout").
			WithTimeout(1 * time.Second). // specify a timeout for the test case
			ExpectStatus(200).
			ExpectBody("Hello, world!"),

		// A description is optional
		myURL.GET("/foo").
			WithHeader("Accept", "application/json").
			WithQueryParam("sort", "false").
			WithBody("hello").
			ExpectStatus(200),

		myURL.GET("/foo", "Fetch foo and expect a subset of JSON in response body").
			WithHeader("Accept", "application/json").
			ExpectStatus(201).
			ExpectBody(json.Object{
				"a_string":       "Hello, world!",
				"a_number":       43,
				"another_number": 3.15,
				"a_bool":         false,
			}),

		myURL.GET("/bar?first=foo&second=bar", "Fetch bar specifying a query string directly").
			ExpectStatus(404),

		myHandler.GET("/foo", "Fetch foo by testing a local handler").
			ExpectStatus(200).
			ExpectBody("Hello, world!"),

		myURL.GET("/bar", "Fetch bar specifying query parameters all at once").
			WithQueryParams(url.Values{
				"first":  []string{"foo"},
				"second": []string{"bar"},
			}).
			ExpectStatus(404),

		myURL.GET("/bar", "Fetch bar specifying query parameters individually").
			WithQueryParam("first", "foo").
			WithQueryParam("second", "bar").
			ExpectStatus(404),

		myURL.POST("/foo", "Create a new foo specifying a Go map as the body").
			WithHeader("Accept", "application/json"). // add a single header
			WithBody(map[string]interface{}{          // specify the body using Go types
				"key": "value",
			}).
			ExpectStatus(201).
			ExpectHeader("My-Custom-Header", "foobar"),

		myHandler.DELETE("/bar", "Delete a bar with a failed expectation").
			ExpectStatus(200),

		myURL.POST("/foo", "Create a foo setting multiple headers at once").
			WithHeaders(http.Header{ // set all headers at once
				"Accept": []string{"application/json"},
				"Auth":   []string{"Bearer 12345"},
			}).
			WithBody(`{"key":"value"}`). // specify body as a string
			ExpectStatus(201),

		myURL.DELETE("/foo", "Delete a foo").
			ExpectStatus(204),

		// use any custom *http.Request for a test
		myURL.DO(customReq, "Fetch foo using a custom HTTP request").
			ExpectStatus(200).
			ExpectBody("Hello, world!"),

		// load expectations from a golden file
		myURL.GET("/foo", "Fetch foo and match expectations from a golden file").
			WithHeader("Accept", "application/json").
			ExpectGolden("golden/expect-headers-and-json-body.golden"),
	})

	if err != nil {
		log.Fatal(err)
	}

	// Results are accessible via the TestResult interface.
	//
	// Setting MELATONIN_OUTPUT=none in the environment will suppress all output
	// to stdout, enabling custom results processing, analysis, and output.
	//
	for _, result := range results {
		fmt.Fprint(io.Discard, result)
	}
}

// Simple static webserver for example purposes.
func startFullExampleServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/foo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Some-Header", "foo")
		w.Header().Add("Some-Header", "bar")

		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			return
		} else if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		body := "Hello, world!"
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			body = `{"a_string":"Hello, world!","a_number":42,"another_number":3.14,"a_bool":true}`
		}

		w.Write([]byte(body))
	})

	return httptest.NewServer(mux)
}
