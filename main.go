package main

import (
	"context"
	"crypto/tls"
	"flag"
	"github.com/elazarl/goproxy"
	"l6p.io/proxy/pkg/cfg"
	"l6p.io/proxy/pkg/logs"
	"l6p.io/proxy/pkg/sys"
	"log"
	"net/http"
	"net/http/httptrace"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type TraceHookType int32
type TraceDataType map[TraceHookType]time.Time

const (
	DNSStart             TraceHookType = 10
	DNSDone              TraceHookType = 11
	DialStart            TraceHookType = 20
	DialDone             TraceHookType = 21
	TLSHandshakeStart    TraceHookType = 30
	TLSHandshakeDone     TraceHookType = 31
	GetConn              TraceHookType = 40
	GotConn              TraceHookType = 41
	GotFirstResponseByte TraceHookType = 50
)

func newReqWithTraceCtx(req *http.Request, ctx *goproxy.ProxyCtx) *http.Request {
	ctx.UserData = make(TraceDataType)

	trace := &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) {
			ctx.UserData.(TraceDataType)[DNSStart] = time.Now()
		},
		DNSDone: func(httptrace.DNSDoneInfo) {
			ctx.UserData.(TraceDataType)[DNSDone] = time.Now()
		},
		ConnectStart: func(string, string) {
			ctx.UserData.(TraceDataType)[DialStart] = time.Now()
		},
		ConnectDone: func(string, string, error) {
			ctx.UserData.(TraceDataType)[DialDone] = time.Now()
		},
		GetConn: func(string) {
			ctx.UserData.(TraceDataType)[GetConn] = time.Now()
		},
		GotConn: func(httptrace.GotConnInfo) {
			ctx.UserData.(TraceDataType)[GotConn] = time.Now()
		},
		GotFirstResponseByte: func() {
			ctx.UserData.(TraceDataType)[GotFirstResponseByte] = time.Now()
		},
		TLSHandshakeStart: func() {
			ctx.UserData.(TraceDataType)[TLSHandshakeStart] = time.Now()
		},
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			ctx.UserData.(TraceDataType)[TLSHandshakeDone] = time.Now()
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	return req
}

func main() {
	verbose := flag.Bool("v", false, "make the operation more talkative")
	addr := flag.String("addr", ":3210", "proxy listen address")
	flag.Parse()

	name := flag.Arg(0)
	config := &cfg.Config{}
	cfg.LoadConfigFromEnv(config)
	logs.InitLoggers(config)

	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = *verbose

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	certificates, err := cfg.InitCertificates(config, tlsConfig.Certificates)
	if err != nil {
		log.Fatalf("Unable to load certificate: %v", err)
	}
	tlsConfig.Certificates = certificates

	goproxy.MitmConnect.TLSConfig = func(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
		return tlsConfig, nil
	}

	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		return newReqWithTraceCtx(req, ctx), nil
	})

	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		traceData := &logs.LogTraceData{}

		if !ctx.UserData.(TraceDataType)[DNSDone].IsZero() &&
			!ctx.UserData.(TraceDataType)[DNSStart].IsZero() {
			traceData.DNSDelta = ctx.UserData.(TraceDataType)[DNSDone].
				Sub(ctx.UserData.(TraceDataType)[DNSStart]).Milliseconds()
		}

		if !ctx.UserData.(TraceDataType)[DialDone].IsZero() &&
			!ctx.UserData.(TraceDataType)[DialStart].IsZero() {
			traceData.DialDelta = ctx.UserData.(TraceDataType)[DialDone].
				Sub(ctx.UserData.(TraceDataType)[DialStart]).Milliseconds()
		}

		if !ctx.UserData.(TraceDataType)[TLSHandshakeDone].IsZero() &&
			!ctx.UserData.(TraceDataType)[TLSHandshakeStart].IsZero() {
			traceData.TLSHandshakeDelta = ctx.UserData.(TraceDataType)[TLSHandshakeDone].
				Sub(ctx.UserData.(TraceDataType)[TLSHandshakeStart]).Milliseconds()
		}

		if !ctx.UserData.(TraceDataType)[GotConn].IsZero() &&
			!ctx.UserData.(TraceDataType)[GetConn].IsZero() {
			traceData.ConnectDelta = ctx.UserData.(TraceDataType)[GotConn].
				Sub(ctx.UserData.(TraceDataType)[GetConn]).Milliseconds()
		}

		if !ctx.UserData.(TraceDataType)[GotFirstResponseByte].IsZero() &&
			!ctx.UserData.(TraceDataType)[GotConn].IsZero() {
			traceData.FirstResponseDelta = ctx.UserData.(TraceDataType)[GotFirstResponseByte].
				Sub(ctx.UserData.(TraceDataType)[GotConn]).Milliseconds()
		}

		logs.Log(&logs.LogData{
			Name:      name,
			Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			Url:       resp.Request.URL.String(),
			Method:    resp.Request.Method,
			Status:    resp.StatusCode,
			Trace:     traceData,
		})
		return resp
	})

	server := &http.Server{Addr: *addr, Handler: proxy}

	gracefulShutdown := make(chan struct{})
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)
		<-stop

		for _, action := range sys.ExitAction {
			action()
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownServerTimeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Shutdown error: %v", err)
		}
		close(gracefulShutdown)
	}()

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Listen and serve error: %v", err)
	}
	<-gracefulShutdown
}
