package main

import (
	"flag"
	"strings"
	"net/http"
	"github.com/elazarl/goproxy"
	"crypto/tls"
	"encoding/base64"
	"log"
	"fmt"
	"sync"
)

var (
	certFile     string
	keyFile      string
	bind         string
	httpPort     string
	httpsPort    string
	authUsername string
	authPassword string
)

func main() {
	flag.StringVar(&certFile, "cert", "", "https cert file")
	flag.StringVar(&keyFile, "key", "", "https cert private key")
	flag.StringVar(&bind, "bind", "0.0.0.0", "bind address")
	flag.StringVar(&httpPort, "port", "1080", "http port")
	flag.StringVar(&httpsPort, "https-port", "", "https port")
	flag.StringVar(&authUsername, "auth-username", "", "http basic auth username")
	flag.StringVar(&authPassword, "auth-password", "", "http basic auth password")
	flag.Parse()

	useHttps := certFile != "" && keyFile != "" && httpsPort != ""
	useBasicAuth := authUsername != "" && authPassword != ""

	var handler http.Handler = goproxy.NewProxyHttpServer()

	if useBasicAuth {
		handler = basicAuth(handler.ServeHTTP, func(username, password string) bool {
			return username == authUsername && password == authPassword
		})
	}

	handler = logRequest(handler)

	wait := &sync.WaitGroup{}

	if httpPort != "" {
		// listen http
		listen := fmt.Sprintf("%s:%s", bind, httpPort)
		server := http.Server{
			Addr:    listen,
			Handler: handler,
		}

		wait.Add(1)
		log.Printf("http listen at %s\n", listen)
		go func() {
			err := server.ListenAndServe()
			if err != nil {
				log.Println(err)
			}
			wait.Done()
		}()
	}

	if useHttps {
		// listen https
		listen := fmt.Sprintf("%s:%s", bind, httpsPort)
		server := http.Server{
			Addr:         listen,
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
			Handler:      handler,
		}

		wait.Add(1)
		log.Printf("https listen at %s\n", listen)
		go func() {
			err := server.ListenAndServeTLS(certFile, keyFile)
			if err != nil {
				log.Println(err)
			}
			wait.Done()
		}()
	}

	wait.Wait()
}

type validator func(username, password string) bool

func basicAuth(pass http.HandlerFunc, validator validator) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		auth := strings.SplitN(r.Header.Get("Proxy-Authorization"), " ", 2)

		if len(auth) != 2 || auth[0] != "Basic" {
			w.Header().Add("Proxy-Authenticate", "Basic realm=Need Username and Password")
			http.Error(w, "require basic proxy auth", http.StatusProxyAuthRequired)
			return
		}

		payload, _ := base64.StdEncoding.DecodeString(auth[1])
		pair := strings.SplitN(string(payload), ":", 2)

		if len(pair) != 2 || !validator(pair[0], pair[1]) {
			http.Error(w, "authorization failed", http.StatusForbidden)
			return
		}

		pass(w, r)
	}
}

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}
