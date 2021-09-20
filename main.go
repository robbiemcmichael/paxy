package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/robbiemcmichael/paxy/internal"
	"github.com/robbiemcmichael/paxy/pkg/proxy"
)

type Server struct {
	Proxy  *proxy.Proxy
	Source []byte
}

func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Host == "" && r.URL.Path == "/pac" {
		w.Write([]byte(server.Source))
		w.Write([]byte("\n"))
	} else {
		server.Proxy.ServeHTTP(w, r)
	}
}

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	log.SetReportCaller(false)

	formatter := new(log.TextFormatter)
	formatter.FullTimestamp = true
	formatter.TimestampFormat = "15:04:05"
	log.SetFormatter(formatter)
}

func usage() {
	fmt.Printf("Usage: %s [options] pac_file\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	port := flag.Int("p", 8228, "The port on which the server listens")

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	pac := internal.PAC{
		File: flag.Arg(0),
	}

	src, err := ioutil.ReadFile(pac.File)

	if err != nil {
		log.Fatalf("Failed to read PAC: %s", err)
	}

	if err := pac.InitWithBytes(src); err != nil {
		log.Fatalf("Failed to initialise PAC: %s", err)
	}

	paxy := &proxy.Proxy{
		Forward: pac.Evaluate,
	}

	if err := paxy.Init(); err != nil {
		log.Fatalf("Failed to initialise proxy: %s", err)
	}

	server := &Server{
		Source: src,
		Proxy:  paxy,
	}

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	log.Fatal(http.ListenAndServe(addr, server))
}
