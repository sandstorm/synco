package receive

import (
	"bytes"
	"encoding/json"
	"errors"
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
	"sync"
	"time"
)

type State string

type ReceiveSession struct {
	baseUrl    *string
	identifier string
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
		// must be a long timeout, otherwise big files cannnot be downloaded.
		Timeout: 3600 * time.Second,

		// we do not want to follow redirects, so that f.e. we do not follow a HTTP to HTTPS redirect.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &client
}

func NewSession(identifier string, password string) (*ReceiveSession, error) {
	workDir := "dump"
	err := os.MkdirAll(workDir, 0755)
	identity, err := age.NewScryptIdentity(password)
	if err != nil {
		return nil, err
	}
	rs := &ReceiveSession{
		baseUrl:    nil,
		identifier: identifier,
		workDir:    &workDir,
		password:   password,
		identity:   identity,
		httpClient: newHttpClient(),
	}

	return rs, nil
}

var ErrMetaFileNotFound = errors.New("file " + dto.FILENAME_META + " not found")

func (rs *ReceiveSession) DoesMetaFileExistOnServer() error {
	resp, err := rs.loadMetaFile()
	if err != nil {
		return err
	}
	// prevent resource leaks
	defer func() { _ = resp.Body.Close() }()

	// meta file exists, no error
	return nil
}

func (rs *ReceiveSession) FetchMeta() (*dto.Meta, error) {
	resp, err := rs.loadMetaFile()
	if err != nil {
		return nil, err
	}
	// prevent resource leaks
	defer func() { _ = resp.Body.Close() }()

	decryptedReader, err := age.Decrypt(resp.Body, rs.identity)
	if err != nil {
		return nil, fmt.Errorf("error decrypting file from server - most likely, the encryption key was wrong: %w", err)
	}
	decoder := json.NewDecoder(decryptedReader)
	meta := &dto.Meta{}
	err = decoder.Decode(&meta)
	if err != nil {
		return nil, fmt.Errorf("Error decrypting file from server (2): %w", err)
	}

	return meta, nil
}

func (rs *ReceiveSession) loadMetaFile() (*http.Response, error) {
	urlToLoad, err := url.JoinPath(*rs.baseUrl, rs.identifier, dto.FILENAME_META)
	pterm.Debug.Printfln("Trying to download %s", urlToLoad)
	if err != nil {
		return nil, err
	}

	resp, err := rs.httpClient.Get(urlToLoad)
	if err != nil {
		pterm.Debug.Printfln("error trying to load %s: %s", urlToLoad, err)
		return nil, ErrMetaFileNotFound
	}
	if resp.StatusCode != 200 {
		pterm.Debug.Printfln("error trying to load %s - wrong status code: %d", urlToLoad, resp.StatusCode)
		// prevent resource leaks
		_ = resp.Body.Close()
		return nil, ErrMetaFileNotFound
	}
	return resp, nil
}
func (rs *ReceiveSession) FetchAndDecryptFileWithProgressBar(fileName string) ([]byte, error) {
	urlToLoad, err := url.JoinPath(*rs.baseUrl, rs.identifier, fileName)
	pterm.Debug.Printfln("Trying to download %s", urlToLoad)
	if err != nil {
		return nil, err
	}

	resp, err := rs.httpClient.Get(urlToLoad)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		pterm.Debug.Printfln("error trying to load %s - wrong status code: %d", urlToLoad, resp.StatusCode)
		// prevent resource leaks
		_ = resp.Body.Close()
		return nil, ErrMetaFileNotFound
	}

	// prevent resource leaks
	defer func() { _ = resp.Body.Close() }()

	downloadByteCounter := &progressbarWriter{}

	downloadByteCounter.pb, _ = pterm.DefaultProgressbar.WithTotal(int(resp.ContentLength)).Start()
	pipeReader, pipeWriter := io.Pipe()

	// we need to call io.Copy in a goroutine; in order to not block forever.
	// NOTE: to catch the error which might happen inside here,
	// we use a WaitGroup to wait for goroutine termination at the end of the method; and additionally
	// check the error then.
	var wg sync.WaitGroup
	var ioCopyErr error
	wg.Add(1)
	go func() {
		_, ioCopyErr = io.Copy(pipeWriter, io.TeeReader(resp.Body, downloadByteCounter))
		_ = pipeWriter.Close()
		wg.Done()
	}()

	if err != nil {
		return nil, err
	}

	decryptedReader, err := age.Decrypt(pipeReader, rs.identity)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(decryptedReader)

	// Now, we can check for ioCopyErr from the goroutine above
	wg.Wait()
	if err != nil {
		return nil, fmt.Errorf("Error decrypting file from server (1): %w - ioCopy err: %s", err, ioCopyErr)
	}

	return buf.Bytes(), nil
}

func (rs *ReceiveSession) FetchFileWithProgressBar(fileName string, progress *pterm.ProgressbarPrinter) ([]byte, error) {
	urlToLoad, err := url.JoinPath(*rs.baseUrl, rs.identifier, fileName)
	pterm.Debug.Printfln("Trying to download %s", urlToLoad)
	if err != nil {
		return nil, err
	}

	resp, err := rs.httpClient.Get(urlToLoad)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		pterm.Debug.Printfln("error trying to load %s - wrong status code: %d", urlToLoad, resp.StatusCode)
		// prevent resource leaks
		_ = resp.Body.Close()
		return nil, ErrMetaFileNotFound
	}

	// prevent resource leaks
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Response status code for %s is %d", urlToLoad, resp.StatusCode)
	}

	downloadByteCounter := &progressbarWriter{}

	downloadByteCounter.pb = progress
	pipeReader, pipeWriter := io.Pipe()

	// we need to call io.Copy in a goroutine; in order to not block forever.
	// NOTE: to catch the error which might happen inside here,
	// we use a WaitGroup to wait for goroutine termination at the end of the method; and additionally
	// check the error then.
	var wg sync.WaitGroup
	var ioCopyErr error
	wg.Add(1)
	go func() {
		_, ioCopyErr = io.Copy(pipeWriter, io.TeeReader(resp.Body, downloadByteCounter))
		_ = pipeWriter.Close()
		wg.Done()
	}()

	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(pipeReader)
	if err != nil {
		return nil, fmt.Errorf("Error reading file from server (1): %w", err)
	}

	// Now, we can check for ioCopyErr from the goroutine above
	wg.Wait()
	if ioCopyErr != nil {
		return nil, fmt.Errorf("Error reading file from server (io.copy): %w", ioCopyErr)
	}

	return buf.Bytes(), nil
}

func (rs *ReceiveSession) DumpAndDecryptFileWithProgressBar(remoteFileName string, localFileName string) error {
	contents, err := rs.FetchAndDecryptFileWithProgressBar(remoteFileName)
	if err != nil {
		return err
	}
	workdirFilePath := rs.filepathInWorkDir(localFileName)
	err = os.MkdirAll(filepath.Dir(workdirFilePath), 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(workdirFilePath, contents, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (rs *ReceiveSession) DumpFileWithProgressBar(remoteFileName string, localFileName string, progress *pterm.ProgressbarPrinter) error {
	contents, err := rs.FetchFileWithProgressBar(remoteFileName, progress)
	if err != nil {
		return err
	}
	workdirFilePath := rs.filepathInWorkDir(localFileName)
	err = os.MkdirAll(filepath.Dir(workdirFilePath), 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(workdirFilePath, contents, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (rs *ReceiveSession) filepathInWorkDir(fileName string) string {
	return filepath.Join(*rs.workDir, fileName)
}

func (rs *ReceiveSession) FileContentsInWorkDir(fileName string) ([]byte, error) {
	return os.ReadFile(rs.filepathInWorkDir(fileName))
}

func (rs *ReceiveSession) SetMTimeInWorkDir(fileName string, mtime time.Time) error {
	err := os.Chtimes(rs.filepathInWorkDir(fileName), mtime, mtime)
	if err != nil {
		return fmt.Errorf("error on setting modification times of %s: %w", fileName, err)
	}
	return nil
}

func (rs *ReceiveSession) StatInWorkDir(fileName string) (os.FileInfo, error) {
	return os.Stat(rs.filepathInWorkDir(fileName))
}

func (rs *ReceiveSession) BaseUrl(baseUrl string) {
	rs.baseUrl = &baseUrl
}

func (rs *ReceiveSession) MetaUrlRelativeToBaseUrl() string {
	return rs.identifier + "/" + dto.FILENAME_META
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
