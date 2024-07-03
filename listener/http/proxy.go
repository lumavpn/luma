package http

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	_ "unsafe"

	"github.com/lumavpn/luma/adapter"
	lru "github.com/lumavpn/luma/common/cache"
	N "github.com/lumavpn/luma/common/net"
	authStore "github.com/lumavpn/luma/listener/auth"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/proxy/inbound"
)

//go:linkname registerOnHitEOF net/http.registerOnHitEOF
func registerOnHitEOF(rc io.ReadCloser, fn func())

//go:linkname requestBodyRemains net/http.requestBodyRemains
func requestBodyRemains(rc io.ReadCloser) bool

func HandleConn(c net.Conn, tunnel adapter.TransportHandler, cache *lru.LruCache[string, bool], additions ...inbound.Addition) {
	client := newClient(c, tunnel, additions...)
	defer client.CloseIdleConnections()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	peekMutex := sync.Mutex{}

	conn := N.NewBufferedConn(c)

	keepAlive := true
	trusted := cache == nil // disable authenticate if lru is nil

	for keepAlive {
		peekMutex.Lock()
		request, err := ReadRequest(conn.Reader())
		peekMutex.Unlock()
		if err != nil {
			break
		}

		request.RemoteAddr = conn.RemoteAddr().String()

		keepAlive = strings.TrimSpace(strings.ToLower(request.Header.Get("Proxy-Connection"))) == "keep-alive"

		var resp *http.Response

		if !trusted {
			var user string
			resp, user = authenticate(request, cache)
			additions = append(additions, inbound.WithInUser(user))
			trusted = resp == nil
		}

		if trusted {
			if request.Method == http.MethodConnect {
				// Manual writing to support CONNECT for http 1.0 (workaround for uplay client)
				if _, err = fmt.Fprintf(conn, "HTTP/%d.%d %03d %s\r\n\r\n", request.ProtoMajor, request.ProtoMinor, http.StatusOK, "Connection established"); err != nil {
					break // close connection
				}

				tunnel.HandleTCPConn(inbound.NewHTTPS(request, conn, additions...))

				return // hijack connection
			}

			host := request.Header.Get("Host")
			if host != "" {
				request.Host = host
			}

			request.RequestURI = ""

			if isUpgradeRequest(request) {
				handleUpgrade(conn, request, tunnel, additions...)

				return // hijack connection
			}

			removeHopByHopHeaders(request.Header)
			removeExtraHTTPHostPort(request)

			if request.URL.Scheme == "" || request.URL.Host == "" {
				resp = responseWith(request, http.StatusBadRequest)
			} else {
				request = request.WithContext(ctx)

				startBackgroundRead := func() {
					go func() {
						peekMutex.Lock()
						defer peekMutex.Unlock()
						_, err := conn.Peek(1)
						if err != nil {
							cancel()
						}
					}()
				}
				if requestBodyRemains(request.Body) {
					registerOnHitEOF(request.Body, startBackgroundRead)
				} else {
					startBackgroundRead()
				}
				resp, err = client.Do(request)
				if err != nil {
					resp = responseWith(request, http.StatusBadGateway)
				}
			}

			removeHopByHopHeaders(resp.Header)
		}

		if keepAlive {
			resp.Header.Set("Proxy-Connection", "keep-alive")
			resp.Header.Set("Connection", "keep-alive")
			resp.Header.Set("Keep-Alive", "timeout=4")
		}

		resp.Close = !keepAlive

		err = resp.Write(conn)
		if err != nil {
			break // close connection
		}
	}

	_ = conn.Close()
}

func authenticate(request *http.Request, cache *lru.LruCache[string, bool]) (resp *http.Response, u string) {
	authenticator := authStore.Authenticator()
	if inbound.SkipAuthRemoteAddress(request.RemoteAddr) {
		authenticator = nil
	}
	if authenticator != nil {
		credential := parseBasicProxyAuthorization(request)
		if credential == "" {
			resp := responseWith(request, http.StatusProxyAuthRequired)
			resp.Header.Set("Proxy-Authenticate", "Basic")
			return resp, ""
		}

		authed, exist := cache.Load(credential)
		if !exist {
			user, pass, err := decodeBasicProxyAuthorization(credential)
			authed = err == nil && authenticator.Verify(user, pass)
			u = user
			cache.Set(credential, authed)
		}
		if !authed {
			log.Infof("Auth failed from %s", request.RemoteAddr)

			return responseWith(request, http.StatusForbidden), u
		}
	}

	return nil, u
}

func responseWith(request *http.Request, statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Proto:      request.Proto,
		ProtoMajor: request.ProtoMajor,
		ProtoMinor: request.ProtoMinor,
		Header:     http.Header{},
	}
}
