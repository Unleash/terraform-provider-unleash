package provider

import (
	"fmt"
	"net/http"
	"net/http/httputil"
)

func httpClient(debug bool) *http.Client {
	return &http.Client{
		Transport: &debugTransport{
			Transport:   http.DefaultTransport,
			EnableDebug: debug,
		},
	}
}

type debugTransport struct {
	Transport   http.RoundTripper
	EnableDebug bool
}

func (t *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.EnableDebug {
		// Log the request details
		requestDump, _ := httputil.DumpRequestOut(req, true)
		fmt.Printf("Request:\n%s\n\n", requestDump)
	}

	// Make the actual request
	resp, err := t.Transport.RoundTrip(req)

	if err != nil {
		fmt.Printf("Err:\n%s\n\n", err)
		
		// only log the response details in case of error to avoid leaking sensitive data
		if t.EnableDebug && resp != nil {
			responseDump, _ := httputil.DumpResponse(resp, true)
			fmt.Printf("Response:\n%s\n\n", responseDump)
		}
	}

	return resp, err
}
