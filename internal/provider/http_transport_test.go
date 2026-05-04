package provider

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticResponseTransport struct{}

func (t staticResponseTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("ok")),
		Request:    req,
	}, nil
}

func Test_concurrentRequestTransport_limitsRequestsUntilResponseBodyIsClosed(t *testing.T) {
	transport := &concurrentRequestTransport{
		Transport: staticResponseTransport{},
		limit:     make(chan struct{}, 2),
	}
	client := &http.Client{Transport: transport}

	acquired := make(chan *http.Response, 5)
	errs := make(chan error, 5)
	closeBodies := make(chan struct{})
	var closeBodiesOnce sync.Once
	unblockBodies := func() {
		closeBodiesOnce.Do(func() {
			close(closeBodies)
		})
	}
	t.Cleanup(func() {
		unblockBodies()
	})

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			resp, err := client.Get("http://example.com")
			if err != nil {
				errs <- err
				return
			}

			acquired <- resp
			<-closeBodies
			errs <- resp.Body.Close()
		}()
	}

	first := <-acquired
	second := <-acquired

	select {
	case resp := <-acquired:
		require.NoError(t, resp.Body.Close())
		t.Fatal("request was not throttled while prior response bodies were still open")
	case <-time.After(50 * time.Millisecond):
	}

	unblockBodies()
	wg.Wait()
	close(errs)

	require.NoError(t, first.Body.Close())
	require.NoError(t, second.Body.Close())
	for err := range errs {
		assert.NoError(t, err)
	}
	assert.Empty(t, transport.limit)
}
