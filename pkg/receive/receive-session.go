package receive

import (
	"filippo.io/age"
	"fmt"
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/pkg/serve"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type State string

type ReceiveSession struct {
	baseUrl    string
	password   string
	identity   *age.ScryptIdentity
	httpClient *http.Client
}

func newHttpClient() *http.Client {
	//ref: Copy and modify defaults from https://golang.org/src/net/http/transport.go
	//Note: Clients and Transports should only be created once and reused
	transport := http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			// Modify the time to wait for a connection to establish
			Timeout:   1 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	client := http.Client{
		Transport: &transport,
		Timeout:   4 * time.Second,
	}

	return &client
}

func NewSession(baseUrl string, password string) (*ReceiveSession, error) {

	identity, err := age.NewScryptIdentity(password)
	if err != nil {
		return nil, err
	}
	rs := &ReceiveSession{
		baseUrl:    baseUrl,
		password:   password,
		identity:   identity,
		httpClient: newHttpClient(),
	}

	return rs, nil
}

func (rs *ReceiveSession) FetchFrameworkName() (string, error) {
	urlToLoad, err := url.JoinPath(rs.baseUrl, serve.FILENAME_FRAMEWORKNAME)
	pterm.Debug.Printfln("Trying to download %s", urlToLoad)
	if err != nil {
		return "", err
	}

	resp, err := rs.httpClient.Get(urlToLoad)
	if err != nil {
		return "", err
	}
	// prevent resource leaks
	defer func() { _ = resp.Body.Close() }()

	decryptedReader, err := age.Decrypt(resp.Body, rs.identity)
	if err != nil {
		return "", fmt.Errorf("Error decrypting file from server (1): %w", err)
	}
	buf := new(strings.Builder)
	_, err = io.Copy(buf, decryptedReader)
	if err != nil {
		return "", fmt.Errorf("Error decrypting file from server (2): %w", err)
	}

	return buf.String(), nil
}
