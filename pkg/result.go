package pkg

import (
	"fmt"
	"strconv"
	"time"
)

type Result struct {
	StatusCode         int
	ExpectedStatusCode int
	ResponseTime       time.Duration
}

func NewResult(statusCode int, expectedStatusCode int, responseTime time.Duration) *Result {
	return &Result{StatusCode: statusCode, ExpectedStatusCode: expectedStatusCode, ResponseTime: responseTime}
}

type Results struct {
	results   []*Result
	processed bool
	fastest   time.Duration
	slowest   time.Duration
	average   time.Duration
	total     time.Duration
}

func NewResults(results []*Result) Results {
	r := Results{results: results}
	r.process()

	return r
}

func (r *Results) process() {
	r.fastest, r.slowest = r.results[0].ResponseTime, r.results[0].ResponseTime

	for _, result := range r.results {
		if result.ResponseTime < r.fastest {
			r.fastest = result.ResponseTime
		}

		if result.ResponseTime > r.slowest {
			r.slowest = result.ResponseTime
		}

		r.total += result.ResponseTime
	}

	r.average = time.Duration(r.total.Nanoseconds() / int64(len(r.results)))
	r.processed = true
}

func (r *Results) Fastest() time.Duration {
	if !r.processed {
		r.process()
	}

	return r.fastest
}

func (r *Results) Slowest() time.Duration {
	if !r.processed {
		r.process()
	}

	return r.slowest
}

func (r *Results) Average() time.Duration {
	if !r.processed {
		r.process()
	}

	return r.average
}

func (r *Results) TotalRequests() int {
	return len(r.results)
}

func (r *Results) RequestsPerSecond(duration time.Duration) float64 {
	if !r.processed {
		r.process()
	}

	res, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", float64(len(r.results))/duration.Seconds()), 64)
	return res
}
