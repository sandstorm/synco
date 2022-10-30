package receive

import (
	"bytes"
	"encoding/json"
	"filippo.io/age"
	"fmt"
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/pkg/common/dto"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type State string

type ReceiveSession struct {
	baseUrl    string
	workDir    *string
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
	workDir := "dump"
	err := os.MkdirAll(workDir, 0755)
	identity, err := age.NewScryptIdentity(password)
	if err != nil {
		return nil, err
	}
	rs := &ReceiveSession{
		baseUrl:    baseUrl,
		workDir:    &workDir,
		password:   password,
		identity:   identity,
		httpClient: newHttpClient(),
	}

	return rs, nil
}

func (rs *ReceiveSession) FetchMeta() (*dto.Meta, error) {
	urlToLoad, err := url.JoinPath(rs.baseUrl, dto.FILENAME_META)
	pterm.Debug.Printfln("Trying to download %s", urlToLoad)
	if err != nil {
		return nil, err
	}

	resp, err := rs.httpClient.Get(urlToLoad)
	if err != nil {
		return nil, err
	}
	// prevent resource leaks
	defer func() { _ = resp.Body.Close() }()

	decryptedReader, err := age.Decrypt(resp.Body, rs.identity)
	if err != nil {
		return nil, fmt.Errorf("Error decrypting file from server (1): %w", err)
	}
	decoder := json.NewDecoder(decryptedReader)
	meta := &dto.Meta{}
	err = decoder.Decode(&meta)
	if err != nil {
		return nil, fmt.Errorf("Error decrypting file from server (2): %w", err)
	}

	return meta, nil
}

func (rs *ReceiveSession) FetchFileWithProgressBar(fileName string) ([]byte, error) {
	urlToLoad, err := url.JoinPath(rs.baseUrl, fileName)
	pterm.Debug.Printfln("Trying to download %s", urlToLoad)
	if err != nil {
		return nil, err
	}

	resp, err := rs.httpClient.Get(urlToLoad)
	if err != nil {
		return nil, err
	}
	// prevent resource leaks
	defer func() { _ = resp.Body.Close() }()

	downloadByteCounter := &progressbarWriter{}

	downloadByteCounter.pb, _ = pterm.DefaultProgressbar.WithTotal(int(resp.ContentLength)).Start()
	pipeReader, pipeWriter := io.Pipe()

	// we need to call io.Copy in a goroutine; in order to not block forever.
	// NOTE: Not sure how to catch the error here :)
	go func() {
		_, _ = io.Copy(pipeWriter, io.TeeReader(resp.Body, downloadByteCounter))
		pipeWriter.Close()
	}()

	fmt.Println("JAAAA2")
	decryptedReader, err := age.Decrypt(pipeReader, rs.identity)
	//decryptedReader, err := age.Decrypt(resp.Body, rs.identity)
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(decryptedReader)
	if err != nil {
		return nil, fmt.Errorf("Error decrypting file from server (1): %w", err)
	}
	return buf.Bytes(), nil
}

func (rs *ReceiveSession) DumpFileWithProgressBar(remoteFileName string, localFileName string) error {
	contents, err := rs.FetchFileWithProgressBar(remoteFileName)
	if err != nil {
		return err
	}
	fmt.Println("JAAAA")
	err = os.WriteFile(rs.filepathInWorkDir(localFileName), contents, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (rs *ReceiveSession) filepathInWorkDir(fileName string) string {
	return filepath.Join(*rs.workDir, fileName)
}

// progressbarWriter counts the number of bytes written to it and adds those to a progressbar;
// taken from https://github.com/pterm/pterm/blob/016c0b4836eb2d047abd52cdfa2f598765a0340c/putils/download-with-progressbar.go
type progressbarWriter struct {
	Total uint64
	pb    *pterm.ProgressbarPrinter
}

func (w *progressbarWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.Total += uint64(n)
	w.pb.Add(len(p))
	return n, nil
}
