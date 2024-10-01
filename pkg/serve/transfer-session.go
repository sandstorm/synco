package serve

import (
	"encoding/json"
	"filippo.io/age"
	"fmt"
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/v2/pkg/common/dto"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type TransferSession struct {
	Identifier string
	WorkDir    *string
	Password   string
	// Meta data to be written. Call UpdateMetadata() after changing this!!
	Meta      *dto.Meta
	recipient *age.ScryptRecipient
	listen    string

	// HTTP Server instance. Non-nil only if listen is set; and after WithFrameworkAndWebDirectory is called.
	httpSrv *http.Server

	// Termination signals
	sigs      chan os.Signal
	DumpAll   bool
	KeepFiles bool
}

func (ts *TransferSession) WithFrameworkAndWebDirectory(frameworkName string, webDirectory string) error {
	// create working directory (err if does not work)
	workDir := filepath.Join(webDirectory, ts.Identifier)
	err := os.MkdirAll(workDir, 0755)
	if err != nil {
		return err
	}
	ts.WorkDir = &workDir

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

func NewSession(identifier string, password string, listen string, all bool, keep bool, sigs chan os.Signal) (*TransferSession, error) {
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
		Identifier: "synco-" + identifier,
		Password:   password,
		DumpAll:    all,
		KeepFiles:  keep,
		recipient:  recipient,
		listen:     listen,
		sigs:       sigs,
	}

	go func() {
		<-sigs
		pterm.Info.Printfln("Cleaning up...")
		_ = os.RemoveAll(*m.WorkDir)
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
	return filepath.Join(*ts.WorkDir, fileName)
}

// EncryptFileToFile returns file size of encrypted file (and error) if needed
func (ts *TransferSession) EncryptFileToFile(srcFileName string, destFileName string) (uint64, error) {
	file, err := os.Open(srcFileName)
	if err != nil {
		return 0, fmt.Errorf("opening file %s: %w", srcFileName, err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	wc, err := ts.EncryptToFile(destFileName)

	if _, err := io.Copy(wc, file); err != nil {
		return 0, fmt.Errorf("encrypting file (1) %s: %w", srcFileName, err)
	}

	err = wc.Close()
	if err != nil {
		return 0, fmt.Errorf("encrypting file (2) %s: %w", srcFileName, err)
	}

	return wc.Size(), nil
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

func (ts *TransferSession) RenderConnectCommand() {
	pterm.Success.Printfln("READY: Execute the following command on the target system to download the dump:")
	pterm.Success.Printfln("")
	pterm.Success.Printfln("          # locally: ")
	pterm.Success.Printfln("          synco receive %s %s", ts.Identifier, ts.Password)
	pterm.Success.Printfln("")
	pterm.Success.Printfln("          # on another server:")
	pterm.Success.Printfln("          curl https://sandstorm.github.io/synco/synco | sh -s - receive %s %s", ts.Identifier, ts.Password)
	pterm.Success.Printfln("")

	if !ts.KeepFiles {
		pterm.Success.Printfln("When you are finished, stop the server by pressing Ctrl-C")
		pterm.Success.Printfln("to have synco clean up your files.")
	} else {
		pterm.Success.Printfln("You are finished.")
		pterm.Warning.Printfln("Syno will --keep the file '%s'.", *ts.WorkDir)
		pterm.Warning.Printfln("You will have to remove it manually!!!")
	}
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
