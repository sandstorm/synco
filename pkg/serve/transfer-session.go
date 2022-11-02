package serve

import (
	"encoding/json"
	"filippo.io/age"
	"fmt"
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/pkg/common/dto"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type TransferSession struct {
	Identifier string
	workDir    *string
	Password   string
	// Meta data to be written. Call UpdateMetadata() after changing this!!
	Meta      *dto.Meta
	recipient *age.ScryptRecipient
	listen    string

	// HTTP Server instance. Non-nil only if listen is set; and after WithFrameworkAndWebDirectory is called.
	httpSrv *http.Server

	// Termination signals
	sigs chan os.Signal
}

func (ts *TransferSession) WithFrameworkAndWebDirectory(frameworkName string, webDirectory string) error {
	// create working directory (err if does not work)
	workDir := filepath.Join(webDirectory, ts.Identifier)
	err := os.MkdirAll(workDir, 0755)
	if err != nil {
		return err
	}
	ts.workDir = &workDir

	if len(ts.listen) > 0 {
		// the user requested to start a HTTP server as well.
		mux := http.NewServeMux()
		mux.Handle("/", http.FileServer(http.Dir(webDirectory)))
		ts.httpSrv = &http.Server{Addr: ts.listen, Handler: mux}
		go func() {
			_ = ts.httpSrv.ListenAndServe()
		}()
	}

	// write that we are ready.
	ts.Meta.FrameworkName = frameworkName
	ts.Meta.State = dto.STATE_INITIALIZING
	err = ts.UpdateMetadata()
	if err != nil {
		return err
	}

	return nil
}

func NewSession(identifier string, password string, listen string, sigs chan os.Signal) (*TransferSession, error) {
	if len(password) == 0 {
		return nil, fmt.Errorf("empty password")
	}

	if len(identifier) == 0 {
		return nil, fmt.Errorf("empty identifier")
	}
	recipient, err := age.NewScryptRecipient(password)
	if err != nil {
		return nil, err
	}
	m := &TransferSession{
		Meta: &dto.Meta{
			State: dto.STATE_CREATED,
		},
		Identifier: "ts-" + identifier,
		Password:   password,
		recipient:  recipient,
		listen:     listen,
		sigs:       sigs,
	}

	go func() {
		<-sigs
		pterm.Info.Printfln("Cleaning up...")
		_ = os.RemoveAll(*m.workDir)
		pterm.Info.Printfln("Cleanup Completed.")
		os.Exit(0)
	}()

	return m, nil
}

const tempSuffix = ".tmp"

func (ts *TransferSession) UpdateMetadata() error {
	// first transfer to temporary file, and then rename atomically to prevent race conditions.
	wc, err := ts.EncryptToFile(dto.FILENAME_META + tempSuffix)
	encoder := json.NewEncoder(wc)
	err = encoder.Encode(ts.Meta)
	if err != nil {
		return err
	}
	err = wc.Close()
	if err != nil {
		return err
	}

	err = os.Rename(ts.filepathInWorkDir(dto.FILENAME_META+tempSuffix), ts.filepathInWorkDir(dto.FILENAME_META))
	if err != nil {
		return err
	}

	// needed for testcases - must run at the END of UpdateMetadata
	return os.WriteFile(ts.filepathInWorkDir("state"), []byte(ts.Meta.State), 0644)
}

func (ts *TransferSession) EncryptBytesToFile(fileName string, contents []byte) error {
	wc, err := ts.EncryptToFile(fileName)
	_, err = wc.Write(contents)
	if err != nil {
		return err
	}
	err = wc.Close()
	if err != nil {
		return err
	}
	return nil
}

func (ts *TransferSession) filepathInWorkDir(fileName string) string {
	return filepath.Join(*ts.workDir, fileName)
}

func (ts *TransferSession) EncryptToFile(fileName string) (WriteCloserWithSize, error) {
	targetFile, err := os.OpenFile(ts.filepathInWorkDir(fileName), os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	// TODO: PASSWORD - RANDOM STRING !?!??! -> OR PUB / PRIV KEY??
	ageWriteCloser, err := age.Encrypt(targetFile, ts.recipient)
	if err != nil {
		return nil, err
	}
	// ageEncryptWriteCloserHelper also closes the targetFile once the WriteCloser gets the close() signal.
	return &ageEncryptWriteCloserHelper{
		f:  targetFile,
		wc: &ageWriteCloser,
	}, nil
}

type WriteCloserWithSize interface {
	io.WriteCloser
	Size() uint64
}

// HELPERS for EncryptToFile. Not sure whether there is a more elegant way of doing this (feels kinda hacky),
// but what we do here is a "hook" to implement when "Close()" is called on the WriteCloser
type ageEncryptWriteCloserHelper struct {
	f            *os.File
	wc           *io.WriteCloser
	writtenBytes uint64
}

func (f *ageEncryptWriteCloserHelper) Size() uint64 {
	return f.writtenBytes
}

func (f *ageEncryptWriteCloserHelper) Write(p []byte) (n int, err error) {
	f.writtenBytes += uint64(len(p))
	return (*f.wc).Write(p)
}

func (f *ageEncryptWriteCloserHelper) Close() error {
	err := (*f.wc).Close()
	if err != nil {
		return err
	}
	// we want to ensure that the target file (f.f) is also closed.
	return f.f.Close()
}
