package provider

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"sync"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func httpClient(debug bool, maxConcurrentRequests int) *http.Client {
	return &http.Client{
		Transport: &concurrentRequestTransport{
			Transport: newDebugTransport(debug),
			limit:     make(chan struct{}, maxConcurrentRequests),
		},
	}
}

type concurrentRequestTransport struct {
	Transport http.RoundTripper
	limit     chan struct{}
}

func (t *concurrentRequestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	select {
	case t.limit <- struct{}{}:
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}

	resp, err := t.Transport.RoundTrip(req)
	if err != nil || resp == nil || resp.Body == nil {
		t.release()
		return resp, err
	}

	resp.Body = &releaseOnCloseReadCloser{
		ReadCloser: resp.Body,
		release:    t.release,
	}

	return resp, err
}

func (t *concurrentRequestTransport) release() {
	<-t.limit
}

type releaseOnCloseReadCloser struct {
	io.ReadCloser
	release func()
	once    sync.Once
}

func (r *releaseOnCloseReadCloser) Close() error {
	var err error
	r.once.Do(func() {
		err = r.ReadCloser.Close()
		r.release()
	})
	return err
}

func newDebugTransport(debug bool) http.RoundTripper {
	return &debugTransport{
		Transport:   http.DefaultTransport,
		EnableDebug: debug,
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
		tflog.Debug(req.Context(), fmt.Sprintf("Request:\n%s", requestDump))
	}

	// Make the actual request
	resp, err := t.Transport.RoundTrip(req)

	if err != nil {
		tflog.Error(req.Context(), err.Error())
	}
	if t.EnableDebug {
		// Log the response details
		if resp != nil {
			responseDump, _ := httputil.DumpResponse(resp, true)
			tflog.Debug(req.Context(), fmt.Sprintf("Response:\n%s", responseDump))
		}
	}

	return resp, err
}
