package serve

import (
	"filippo.io/age"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type State string

const (
	STATE_CREATED      State = "Created"
	STATE_INITIALIZING State = "Initializing"
	STATE_READY        State = "Ready"
)

const (
	FILENAME_FRAMEWORKNAME = "frameworkName"
)

type TransferSession struct {
	identifier string
	workDir    *string
	password   string
	state      State
	recipient  *age.ScryptRecipient
	listen     string

	// HTTP Server instance. Non-nil only if listen is set; and after WithFrameworkAndWebDirectory is called.
	httpSrv *http.Server
}

func (m *TransferSession) WithFrameworkAndWebDirectory(frameworkName string, webDirectory string) error {
	// create working directory (err if does not work)
	workDir := filepath.Join(webDirectory, m.identifier)
	err := os.MkdirAll(workDir, 0755)
	if err != nil {
		return err
	}
	m.workDir = &workDir

	if len(m.listen) > 0 {
		// the user requested to start a HTTP server as well.
		mux := http.NewServeMux()
		mux.Handle("/", http.FileServer(http.Dir(webDirectory)))
		m.httpSrv = &http.Server{Addr: m.listen, Handler: mux}
		go func() {
			_ = m.httpSrv.ListenAndServe()
		}()
	}

	// write framework name
	err = m.EncryptStringToFile(FILENAME_FRAMEWORKNAME, frameworkName)
	if err != nil {
		return err
	}

	// write that we are ready.
	err = m.UpdateState(STATE_INITIALIZING)
	if err != nil {
		return err
	}

	return nil
}

func NewSession(identifier string, password string, listen string) (*TransferSession, error) {

	recipient, err := age.NewScryptRecipient(password)
	if err != nil {
		return nil, err
	}
	m := &TransferSession{
		state:      STATE_CREATED,
		identifier: identifier,
		password:   password,
		recipient:  recipient,
		listen:     listen,
	}

	return m, nil
}

func (m *TransferSession) UpdateState(state State) error {
	m.state = state

	err := os.WriteFile(filepath.Join(*m.workDir, "state"), []byte(m.state), 0755)
	if err != nil {
		return err
	}

	return nil
}

func (m *TransferSession) EncryptStringToFile(fileName string, contents string) error {
	wc, err := m.EncryptToFile(fileName)
	_, err = wc.Write([]byte(contents))
	if err != nil {
		return err
	}
	err = wc.Close()
	if err != nil {
		return err
	}
	return nil
}

func (m *TransferSession) EncryptToFile(fileName string) (io.WriteCloser, error) {
	targetFile, err := os.OpenFile(filepath.Join(*m.workDir, fileName), os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	// TODO: PASSWORD - RANDOM STRING !?!??! -> OR PUB / PRIV KEY??
	ageWriteCloser, err := age.Encrypt(targetFile, m.recipient)
	if err != nil {
		return nil, err
	}
	// ageEncryptWriteCloserHelper also closes the targetFile once the WriteCloser gets the close() signal.
	return ageEncryptWriteCloserHelper{
		f:  targetFile,
		wc: &ageWriteCloser,
	}, nil
}

// HELPERS for EncryptToFile
type ageEncryptWriteCloserHelper struct {
	f  *os.File
	wc *io.WriteCloser
}

func (f ageEncryptWriteCloserHelper) Write(p []byte) (n int, err error) {
	return (*f.wc).Write(p)
}

func (f ageEncryptWriteCloserHelper) Close() error {
	err := (*f.wc).Close()
	if err != nil {
		return err
	}
	// we want to ensure that the target file (f.f) is also closed.
	return f.f.Close()
}
