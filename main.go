package main

import (
	"context"
	"fmt"
	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uiprogress/util/strutil"
	"github.com/urfave/cli"
	"go.ajitem.com/chopper/pkg"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	concurrency, maxRedirects int
	duration                  time.Duration
	headers                   http.Header
	location, cookies         bool
	r                         = make(chan *pkg.Result)
	done                      = make(chan bool)
	rslt                      []*pkg.Result
	err                       error
)

func main() {
	app := cli.NewApp()

	app.Name = "chopper"
	app.Usage = "modern utility for http benchmarking"
	app.Copyright = "(c) 2019 Ajitem Sahasrabuddhe"

	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "concurrency, c",
			Usage: "Number of requests to perform at once.",
			Value: 1,
		},
		cli.StringFlag{
			Name:  "duration, d",
			Usage: "Time through which to run the benchmark. Accepts units of time s - second, m - minute, h - hour",
			Value: "1h",
		},
		cli.StringSliceFlag{
			Name:  "header, H",
			Usage: "Append an extra header to the request",
		},
		cli.StringFlag{
			Name:  "useragent, A",
			Usage: "Specify the User-Agent string to send to the HTTP server.",
		},
		cli.StringFlag{
			Name:  "ip-address, i",
			Usage: "Specify an IP Address to send to the HTTP server as the source of the request.",
		},
		cli.BoolFlag{
			Name:  "keep-alive, k",
			Usage: "Makes chopper reuse existing connections for subsequent requests",
		},
		cli.StringFlag{
			Name:  "request, X",
			Usage: "Specify a request method to use when communicating with the HTTP server. Supported requests: GET, POST, PUT, DELETE",
			Value: "GET",
		},
		cli.BoolFlag{
			Name:  "location, L",
			Usage: "Follow the location header in case of a 3xx Response code",
		},
		cli.IntFlag{
			Name:  "max-redirects",
			Usage: "Number of redirects to follow before stopping.",
			Value: 0,
		},
		cli.BoolFlag{
			Name:  "cookie-jar, C",
			Usage: "Specify a whether or not to use cookies",
		},
		cli.StringFlag{
			Name:     "url, u",
			Usage:    "URL to benchmark",
			Required: true,
		},
	}

	app.Action = ActionFunc

	app.Setup()
	app.Commands = cli.Commands{}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func ActionFunc(c *cli.Context) {
	// read concurrency
	concurrency = c.Int("concurrency")
	// read and parse duration
	duration, err = time.ParseDuration(c.String("duration"))
	if err != nil {
		log.Println("incorrect value of duration supplied:", err)
		_ = cli.ShowAppHelp(c)
	}

	// set headers
	headers = http.Header{}
	for _, header := range c.StringSlice("header") {
		h := strings.Split(header, ":")
		if len(h) == 2 {
			headers.Add(strings.TrimSpace(h[0]), strings.TrimSpace(h[1]))
		}
	}

	// set useragent if specified
	userAgent := c.String("useragent")
	if userAgent != "" {
		headers.Add("User-Agent", userAgent)
	}

	// set ip address if specified
	ipAddress := net.ParseIP(c.String("ip-address"))
	if ipAddress != nil {
		headers.Add("X-Forwarded-For", ipAddress.String())
	}

	// set the connection header
	if c.Bool("keep-alive") {
		headers.Add("Connection", "keep-alive")
	}

	// follow redirects ?
	location = c.Bool("location")
	// how many redirects to follow ?
	maxRedirects = c.Int("max-redirects")

	// use cookies ?
	cookies = c.Bool("cookie-jar")

	// prepare to start the firing, create new request
	req, err := http.NewRequest(c.String("request"), c.String("url"), nil)
	if err != nil {
		log.Fatal(err)
	}

	// add headers
	req.Header = headers

	request, _ := pkg.NewRequest(req, location, maxRedirects, cookies)

	// setup receiver
	go func() {
		for res := range r {
			rslt = append(rslt, res)
		}
		done <- true
	}()

	// setup context
	start := time.Now()
	ctx, _ := context.WithTimeout(context.Background(), duration)
	go func() {
		select {
		case <-ctx.Done():
			close(r)
		}
	}()

	uiprogress.Start()
	for i := 0; i < concurrency; i++ {
		go func(request *pkg.Request, index int) {
			bar := uiprogress.AddBar(int(duration.Microseconds())).AppendCompleted()
			bar.TimeStarted = start
			bar.PrependFunc(func(b *uiprogress.Bar) string {
				return fmt.Sprintf("%s %s", strutil.PadLeft(fmt.Sprintf("worker %3d", index), 5, ' '), strutil.PadLeft(strutil.PrettyTime(time.Since(start)), 1, ' '))
			})

			for {
				r <- request.Do(0)

				elapsed := time.Since(start).Microseconds()
				_ = bar.Set(int(elapsed))
			}
		}(request, i)
	}

	<-done

	results := pkg.NewResults(rslt)

	fmt.Println("Total requests completed:", results.TotalRequests())
	fmt.Println("Requests per second:", results.RequestsPerSecond(duration))

	fmt.Println("Fastest Request Time:", results.Fastest())
	fmt.Println("Slowest Request Time:", results.Slowest())
	fmt.Println("Average Request Time:", results.Average())
}
