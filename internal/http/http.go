package http

import (
	"github.com/quero-edu/loki-rule-operator/internal/flags"
	"net/http"
)

type WithHeader struct {
	http.Header
	rt http.RoundTripper
}

func ApplyHeader(rt http.RoundTripper) WithHeader {
	if rt == nil {
		rt = http.DefaultTransport
	}

	return WithHeader{Header: make(http.Header), rt: rt}
}

func (h WithHeader) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(h.Header) == 0 {
		return h.rt.RoundTrip(req)
	}

	req = req.Clone(req.Context())
	for k, v := range h.Header {
		req.Header[k] = v
	}

	return h.rt.RoundTrip(req)
}

func ClientWithHeaders(extraHeaders *flags.ArrayFlags) *http.Client {
	client := &http.Client{}

	rt := ApplyHeader(client.Transport)
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
