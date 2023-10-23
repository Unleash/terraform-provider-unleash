package provider

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	tflog.Info(req.Context(), fmt.Sprintf("Enable debug logging %v", t.EnableDebug))
	if t.EnableDebug {
		// Log the request details
		requestDump, _ := httputil.DumpRequestOut(req, true)
		tflog.Debug(req.Context(), fmt.Sprintf("Request:\n%s", requestDump))
	}

	// Make the actual request
	resp, err := t.Transport.RoundTrip(req)

	if err != nil {
		tflog.Error(req.Context(), err.Error())
	}

	// only log the response details in case of error to avoid leaking sensitive data
	if t.EnableDebug && resp != nil && (err != nil || (resp.StatusCode >= 200 && resp.StatusCode < 300)) {
		responseDump, _ := httputil.DumpResponse(resp, true)
		tflog.Debug(req.Context(), fmt.Sprintf("Response:\n%s", responseDump))
	}

	return resp, err
}
