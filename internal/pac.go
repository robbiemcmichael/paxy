package internal

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/jackwakefield/gopac"
	log "github.com/sirupsen/logrus"
)

type PAC struct {
	File   string
	Parser *gopac.Parser
	mux    sync.Mutex
}

func (pac *PAC) Init() error {
	pac.Parser = new(gopac.Parser)
	return pac.Parser.Parse(pac.File)
}

func (pac *PAC) Evaluate(r *http.Request) (*url.URL, error) {
	// gopac isn't thread safe, so we have to use a mutex
	pac.mux.Lock()
	proxy, err := pac.Parser.FindProxy(r.URL.String(), r.URL.Hostname())
	pac.mux.Unlock()

	reqLogger := log.WithFields(
		log.Fields{
			"pacFile": pac.File,
			"proxy":   proxy,
		},
	)

	errCtx := fmt.Sprintf("evaluating PAC file %s", pac.File)

	if err == nil {

		reqLogger.Infof("%s %s", r.Method, r.URL.String())

		fields := strings.Fields(proxy)
		if len(fields) == 0 {
			msg := "empty proxy configuration string"
			reqLogger.Warnf("PAC file error: %s", msg)
			return nil, fmt.Errorf("%s: %s", errCtx, msg)
		}

		switch fields[0] {
		case "DIRECT":
			return nil, nil

		case "PROXY", "HTTP":
			if len(fields) < 2 {
				msg := fmt.Sprintf("expected proxy to have format: %s host:port", fields[0])
				reqLogger.Warnf("PAC file error: %s", msg)
				return nil, fmt.Errorf("%s: expected proxy to have format \"%s host:port\" but got %q", errCtx, fields[0], proxy)
			}

			if len(fields) > 2 {
				reqLogger.Warnf("PAC file returned multiple proxies, only the first will be used")
			}

			target := "http://" + strings.TrimSuffix(fields[1], ";")
			return url.Parse(target)

		case "SOCKS", "SOCKS4", "SOCKS5":
			reqLogger.Warn("PAC file error: unsupported proxy")
			return nil, fmt.Errorf("%s: unsupported proxy: %s", errCtx, proxy)

		default:
			reqLogger.Warn("PAC file error: unrecognised proxy")
			return nil, fmt.Errorf("%s: invalid proxy: %s", errCtx, proxy)
		}
	} else {
		reqLogger.Errorf("PAC file error: %s", err)
		return nil, err
	}

	return nil, nil
}
