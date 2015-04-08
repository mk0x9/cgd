package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"net"
	"net/http"
	"net/http/cgi"
	"net/http/fcgi"
)

var cmd = flag.String("c", "", "CGI program to run")
var pwd = flag.String("w", "", "Working dir for CGI")
var serveFcgi = flag.Bool("f", false, "Run as a FCGI 'server' instead of HTTP")
var debug = flag.Bool("debug", false, "Print debug msgs to stderr")
var address = flag.String("a", ":3333", "Listen address")
var envVars = flag.String("e", "", "Comma-separated list of environment variables to preserve")

var passtroughHeaders = []string{"AUTH_TYPE", "REMOTE_USER"} // set of headers to passthrough from http backend

func main() {
	flag.Usage = usage
	flag.Parse()

	if *cmd == "" {
		usage()
	}

	// This is a hack to make p9p's rc happier for some unknown reason.
	c := *cmd
	if c[0] != '/' {
		c = "./" + c
	}

	os.Setenv("PATH", os.Getenv("PATH")+":.")

	envList := []string{"PATH", "PLAN9"}
	for _, envVar := range strings.Split(*envVars, ",") {
		envList = append(envList, envVar)
	}

	var err error
	if *serveFcgi {
		if l, err := net.Listen("tcp", *address); err == nil {
			log.Println("Starting FastCGI daemon listening on", *address)
			err = fcgi.Serve(l, &cgi.Handler{
				Path:       c,
				Root:       "/",
				Dir:        *pwd,
				InheritEnv: envList,
			})
		}

	} else {
		log.Println("Starting HTTP server listening on", *address)
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			var acc = []string{}
			for header := range r.Header {
				if v := r.Header.Get(header); v != "" {
					for _, pHeader := range passtroughHeaders {
						if strings.ToLower(header) == strings.ToLower(pHeader) {
							acc = append(acc, pHeader+"="+v)
						}
					}
				}
			}
			(&cgi.Handler{
				Path:       c,
				Root:       "/",
				Dir:        *pwd,
				InheritEnv: envList,
				Env:        acc,
			}).ServeHTTP(w, r)
		})
		err = http.ListenAndServe(*address, nil)
	}

	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	os.Stderr.WriteString("usage: cgd -c prog [-w wdir] [-a addr] [-e FOO,BAR]\n")
	flag.PrintDefaults()
	os.Exit(2)
}
