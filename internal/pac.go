package internal

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/luke-clifton/gopac"
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

func (pac *PAC) InitWithBytes(input []byte) error {
	pac.Parser = new(gopac.Parser)
	return pac.Parser.ParseBytes(input)
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

	errCtx := fmt.Sprintf("evaluating PAC file %q", pac.File)

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

		case "PROXY", "HTTP", "SOCKS", "SOCKS5":
			scheme, err := getScheme(fields)
			if err != nil {
				reqLogger.Warnf("Invalid proxy configuration string")
				return nil, fmt.Errorf("%s: proxy configuration string %q: %s", errCtx, proxy, err)
			}

			return url.Parse(scheme + "://" + strings.TrimSuffix(fields[1], ";"))

		case "SOCKS4":
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

func getScheme(fields []string) (string, error) {
	if len(fields) == 0 {
		return "", errors.New("empty proxy configuration string")
	}

	if len(fields) == 1 {
		return "", fmt.Errorf("expected format \"%s host:port\"", fields[0])
	}

	if len(fields) > 2 {
		log.Warnf("PAC file returned multiple proxies, only the first will be used")
	}

	switch fields[0] {
	case "PROXY", "HTTP":
		return "http", nil
	case "SOCKS", "SOCKS5":
		return "socks5", nil
	default:
		return "", fmt.Errorf("invalid proxy type: %s", fields[0])
	}
}
