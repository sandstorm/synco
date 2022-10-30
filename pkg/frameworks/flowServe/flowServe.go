package flowServe

import (
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/pkg/common"
	"github.com/sandstorm/synco/pkg/common/dto"
	"github.com/sandstorm/synco/pkg/serve"
	"github.com/sandstorm/synco/pkg/util"
	"github.com/sandstorm/synco/pkg/util/mysql"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"strings"
)

type flowServe struct {
}

func (f flowServe) Name() string {
	return "Neos/Flow"
}

func (f flowServe) Detect() bool {
	if _, err := os.Stat("flow"); err != nil {
		pterm.Debug.Println("./flow binary not found, thus no installed Flow ServeFramework")
		return false
	}

	if _, err := os.Stat("Web"); err != nil {
		pterm.Debug.Println("./Web folder not found, thus no installed Flow ServeFramework")
		return false
	}

	return true
}

type flowPersistenceBackendOptions struct {
	Driver   string `yaml:"driver"`
	Host     string `yaml:"host"`
	DbName   string `yaml:"dbname"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Charset  string `yaml:"charset"`
	Port     int    `yaml:"port"`
}

func (fp *flowPersistenceBackendOptions) ToDbCredentials() *common.DbCredentials {
	port := 3306
	if fp.Port != 0 {
		port = fp.Port
	}
	return &common.DbCredentials{
		Host:     fp.Host,
		Port:     port,
		User:     fp.User,
		Password: fp.Password,
		DbName:   fp.DbName,
	}
}

func (f flowServe) Serve(transferSession *serve.TransferSession) {
	err := transferSession.WithFrameworkAndWebDirectory(f.Name(), "Web")
	if err != nil {
		pterm.Fatal.Printfln("Error writing transferSession: %s", err)
	}

	pterm.Info.Println("Finding database credentials")

	// 1) DATABASE CREDENTIALS
	// Figure out database credentials
	cmd := exec.Command("./flow", "configuration:show", "--type", "Settings", "--path", "Neos.Flow.persistence.backendOptions")
	output, err := util.RunWrappedCommand(cmd)
	if err != nil {
		pterm.Fatal.Printfln("./flow configuration:show did not succeed: %s", err)
	}
	// remove the first line; as it contains the " Configuration "Settings: Neos.Flow.persistence.backendOptions":" line:
	outputParts := strings.SplitN(output, "\n", 2)
	output = outputParts[1]
	var flowPersistence flowPersistenceBackendOptions
	err = yaml.Unmarshal([]byte(output), &flowPersistence)
	if err != nil {
		pterm.Fatal.Printfln("could not parse output of ./flow configuration:show: %s. Output was: %s", err, output)
	}
	pterm.Success.Printfln("Extracted Database Host %s, User: %s", flowPersistence.Host, flowPersistence.User)

	// 2) DATABASE DUMP
	// basically the way it works is:
	// mysql.CreateDump --> age.Encrypt --> write to file.
	// but because this is based on streams, we need to construct it the other way around:
	// 1st: open the target file
	// 2nd: init age.Encrypt
	// 3rd: do mysql dump (which feeds the Writer)
	wc, err := transferSession.EncryptToFile("dump.sql.enc")
	fileSet := &dto.FileSet{
		Name: "dbDump",
		Type: dto.TYPE_MYSQLDUMP,
		MysqlDump: &dto.FileSetMysqlDump{
			FileName: "dump.sql.enc",
		},
	}

	// 2b) the actual DB dump. also finishes writing.
	err = mysql.CreateDump(flowPersistence.ToDbCredentials(), wc)
	if err != nil {
		pterm.Fatal.Printfln("could not create SQL dump: %s", err)
	}
	fileSet.MysqlDump.SizeBytes = wc.Size()
	transferSession.Meta.FileSets = append(transferSession.Meta.FileSets, fileSet)
	err = transferSession.UpdateMetadata()
	if err != nil {
		pterm.Fatal.Printfln("could not update SQL dump metadata: %s", err)
	}

	pterm.Success.Printfln("Stored Database Dump in %s", "dump.sql.enc")
	transferSession.Meta.State = dto.STATE_READY
	err = transferSession.UpdateMetadata()
	if err != nil {
		pterm.Fatal.Printfln("could not update state: %s", err)
	}
	pterm.Success.Printfln("Ready: synco receive http://your-base-url/%s %s", transferSession.Identifier, transferSession.Password)
}

func NewFlowFramework() common.ServeFramework {
	return &flowServe{}
}
