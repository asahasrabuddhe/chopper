package pkg

import (
	"golang.org/x/net/publicsuffix"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
	"time"
)

type Request struct {
	start           time.Time
	mutex           sync.Mutex
	client          *http.Client
	request         *http.Request
	followRedirects bool
	maxRedirects    int
}

func NewRequest(request *http.Request, followRedirects bool, maxRedirects int, useCookies bool) (*Request, error) {
	if useCookies {
		jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		if err != nil {
			return nil, err
		}

		return &Request{
			client: &http.Client{Jar: jar}, request: request,
			followRedirects: followRedirects, maxRedirects: maxRedirects,
		}, nil
	}

	return &Request{
		client: &http.Client{}, request: request,
		followRedirects: followRedirects, maxRedirects: maxRedirects,
	}, nil
}

func (r *Request) Do(redirectCount int) *Result {
	if redirectCount > r.maxRedirects {
		return nil
	}

	if redirectCount == 0 {
		r.start = time.Now()
	}

	res, err := r.client.Do(r.request)
	if err != nil {
		return nil
	}

	if r.followRedirects && (res.StatusCode >= 300 && res.StatusCode < 400) {
		r.request.URL, _ = url.Parse(res.Header.Get("Location"))
		return r.Do(redirectCount + 1)
	}

	if redirectCount == 0 {
		return NewResult(res.StatusCode, 200, time.Since(r.start))
	}

	return nil
}
