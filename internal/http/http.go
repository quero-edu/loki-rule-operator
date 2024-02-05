package http

import (
	"github.com/quero-edu/loki-rule-operator/internal/flags"
	"net/http"
)

type withHeader struct {
	http.Header
	rt http.RoundTripper
}

func WithHeader(rt http.RoundTripper) withHeader {
	if rt == nil {
		rt = http.DefaultTransport
	}

	return withHeader{Header: make(http.Header), rt: rt}
}

func (h withHeader) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(h.Header) == 0 {
		return h.rt.RoundTrip(req)
	}

	req = req.Clone(req.Context())
	for k, v := range h.Header {
		req.Header[k] = v
	}

	return h.rt.RoundTrip(req)
}

func HttpClientWithHeaders(extraHeaders *flags.ArrayFlags) *http.Client {
	client := &http.Client{}

	rt := WithHeader(client.Transport)
	pairs, err := extraHeaders.Split("=")
	if err != nil {
		panic(err)
	}

	for key, value := range pairs {
		rt.Set(key, value)
	}
	client.Transport = rt

	return client
}
