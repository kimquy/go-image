// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ctxhttp provides helper functions for performing context-aware HTTP requests.
package ctxhttp

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"urlscanner/Godeps/_workspace/src/golang.org/x/net/context"
)

// Do sends an HTTP request with the provided http.Client and returns an HTTP response.
// If the client is nil, http.DefaultClient is used.
// If the context is canceled or times out, ctx.Err() will be returned.
func Do(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	if client == nil {
		client = http.DefaultClient
	}

	// Request cancelation changed in Go 1.5, see cancelreq.go and cancelreq_go14.go.
	cancel := canceler(client, req)

	type responseAndError struct {
		resp *http.Response
		err  error
	}
	result := make(chan responseAndError, 1)

	go func() {
		resp, err := client.Do(req)
		result <- responseAndError{resp, err}
	}()

	var resp *http.Response

	select {
	case <-ctx.Done():
		cancel()
		return nil, ctx.Err()
	case r := <-result:
		var err error
		resp, err = r.resp, r.err
		if err != nil {
			return resp, err
		}
	}

	c := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			cancel()
		case <-c:
			// The response's Body is closed.
		}
	}()
	resp.Body = &notifyingReader{resp.Body, c}

	return resp, nil
}

// Get issues a GET request via the Do function.
func Get(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return Do(ctx, client, req)
}

// Head issues a HEAD request via the Do function.
func Head(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	return Do(ctx, client, req)
}

// Post issues a POST request via the Do function.
func Post(ctx context.Context, client *http.Client, url string, bodyType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	return Do(ctx, client, req)
}

// PostForm issues a POST request via the Do function.
func PostForm(ctx context.Context, client *http.Client, url string, data url.Values) (*http.Response, error) {
	return Post(ctx, client, url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// notifyingReader is an io.ReadCloser that closes the notify channel after
// Close is called or a Read fails on the underlying ReadCloser.
type notifyingReader struct {
	io.ReadCloser
	notify chan<- struct{}
}

func (r *notifyingReader) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if err != nil && r.notify != nil {
		close(r.notify)
		r.notify = nil
	}
	return n, err
}

func (r *notifyingReader) Close() error {
	err := r.ReadCloser.Close()
	if r.notify != nil {
		close(r.notify)
		r.notify = nil
	}
	return err
}
