package flow

import (
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/pkg/frameworks"
	"github.com/sandstorm/synco/pkg/util"
	"github.com/sandstorm/synco/pkg/util/mysql"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"strings"
)

type flowFramework struct {
}

func (f flowFramework) Name() string {
	return "Neos/Flow"
}

func (f flowFramework) Detect() bool {
	if _, err := os.Stat("flow"); err != nil {
		pterm.Debug.Println("./flow binary not found, thus no installed Flow Framework")
		return false
	}

	if _, err := os.Stat("Web"); err != nil {
		pterm.Debug.Println("./Web folder not found, thus no installed Flow Framework")
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

func (fp *flowPersistenceBackendOptions) ToDbCredentials() *frameworks.DbCredentials {
	port := 3306
	if fp.Port != 0 {
		port = fp.Port
	}
	return &frameworks.DbCredentials{
		Host:     fp.Host,
		Port:     port,
		User:     fp.User,
		Password: fp.Password,
		DbName:   fp.DbName,
	}
}

func (f flowFramework) Serve() {
	pterm.Info.Println("Finding database credentials")

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
	pterm.Success.Println("Extracted Database Host %s, User: %s", flowPersistence.Host, flowPersistence.User)
	resultFile, err := mysql.CreateDump(flowPersistence.ToDbCredentials(), "Data/Temporary/", "dbdump")
	if err != nil {
		pterm.Fatal.Printfln("could create SQL dump: %s", err)
	}

	pterm.Success.Printfln("Stored Database Dump in %s", resultFile)
}

func NewFlowFramework() frameworks.Framework {
	return &flowFramework{}
}
