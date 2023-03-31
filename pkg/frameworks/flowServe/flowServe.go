package flowServe

import (
	"encoding/json"
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/pkg/common"
	"github.com/sandstorm/synco/pkg/common/dto"
	"github.com/sandstorm/synco/pkg/serve"
	"github.com/sandstorm/synco/pkg/util"
	"github.com/sandstorm/synco/pkg/util/mysql"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
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

type flowResourceOptions struct {
	Collections map[string]flowResourceCollection `yaml:"collections"`
	Targets     map[string]flowResourceTarget     `yaml:"targets"`
}

func (o *flowResourceOptions) FindPersistentTarget() *flowResourceTarget {
	var persistentTargetName string
	for key, collection := range o.Collections {
		if key == "persistent" {
			pterm.Info.Printfln("collection 'persistent' is using target '%s'", collection.Target)
			persistentTargetName = collection.Target
		}
	}

	if persistentTargetName == "" {
		pterm.Warning.Printfln("did not find collection 'persistent' in config")
		return nil
	}

	for key, target := range o.Targets {
		if key == persistentTargetName {
			pterm.Info.Printfln("target '%s' is configured as follows: '%+v'", persistentTargetName, target)

			return &target
		}
	}

	pterm.Warning.Printfln("did not find persistent target '%s' in config.", persistentTargetName)
	return nil
}

type flowResourceCollection struct {
	// storage does not matter, so we only add target for now
	Target string `yaml:"target"`
}
type flowResourceTarget struct {
	Target        string `yaml:"target"`
	TargetOptions struct {
		// **for Neos\Flow\ResourceManagement\Target\FileSystemSymlinkTarget:**
		// f.e. /Users/sebastian/src/neos-90/Web/_Resources/Persistent/
		Path string `yaml:"path"`
		// f.e. _Resources/Persistent/ - or
		// https://cdn.yourwebsite.de/resources/' for S3Target
		BaseUri string `yaml:"baseUri"`

		// **for Flownative\Aws\S3\S3Target**
		// f.e. prod-neos-cdn
		Bucket string `yaml:"bucket"`
		// f.e. resources/ - see BaseUri
		KeyPrefix string `yaml:"keyPrefix"`
	} `yaml:"targetOptions"`
}

func (t flowResourceTarget) IsS3Target() bool {
	return t.Target == "Flownative\\Aws\\S3\\S3Target"
}

func (t flowResourceTarget) IsFileSystemTarget() bool {
	return t.Target == "Neos\\Flow\\ResourceManagement\\Target\\FileSystemSymlinkTarget" ||
		t.Target == "Neos\\Flow\\ResourceManagement\\Target\\FileSystemTarget"

}

type flowPersistenceBackendOptions struct {
	Driver   string `yaml:"driver"`
	Host     string `yaml:"host"`
	DbName   string `yaml:"dbname"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Charset  string `yaml:"charset"`
	Port     string `yaml:"port"`
}

func (fp *flowPersistenceBackendOptions) ToDbCredentials() *common.DbCredentials {
	port := 3306
	if len(fp.Port) != 0 {
		port, _ = strconv.Atoi(fp.Port)
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

	flowPersistence := f.extractDatabaseCredentialsFromFlow()
	ignoreContentOfTables := []string{
		// event log can be HUGE and is usually not needed.
		"neos_neos_eventlog_domain_model_event",
		// thumbnails can be regenerated
		"neos_media_domain_model_thumbnail",
	}
	if transferSession.DumpAll {
		ignoreContentOfTables = []string{}
	}
	f.databaseDump(transferSession, flowPersistence, ignoreContentOfTables)
	flowResourceConfig := f.extractResourceConfigFromFlow()
	persistentTarget := flowResourceConfig.FindPersistentTarget()

	if persistentTarget == nil {
		pterm.Warning.Printfln("falling back to extracting locations from default location.")
		// fallback to extracting resources from default location
		f.extractResources(transferSession, "./Web/_Resources/Persistent", "_Resources/Persistent")
	} else if persistentTarget.IsS3Target() {
		pterm.Info.Printfln("Extracting resources for S3Target (baseUri=%s)", persistentTarget.TargetOptions.BaseUri)
		pterm.Fatal.Printfln("TODO IMPLEMENT ME")
	} else if persistentTarget.IsFileSystemTarget() {
		pterm.Info.Printfln("Extracting resources for FileSystemTarget (path=%s, baseUri=%s)", persistentTarget.TargetOptions.Path, persistentTarget.TargetOptions.BaseUri)
		f.extractResources(transferSession, persistentTarget.TargetOptions.Path, persistentTarget.TargetOptions.BaseUri)
	} else {
		pterm.Fatal.Printfln("unknown persistent target type '%s'", persistentTarget.Target)
	}

	transferSession.Meta.State = dto.STATE_READY
	err = transferSession.UpdateMetadata()
	if err != nil {
		pterm.Fatal.Printfln("could not update state: %s", err)
	}
	pterm.Success.Printfln("")
	pterm.Success.Printfln("=================================================================================")
	pterm.Success.Printfln("")
	if len(ignoreContentOfTables) > 0 {
		pterm.Success.Printfln("The dump does NOT contain:")
		for _, table := range ignoreContentOfTables {
			pterm.Success.Printfln("- %s", table)
		}
		pterm.Success.Printfln("")
		pterm.Success.Printfln("In case you want to dump all tables, run with --all.")
	}

	pterm.Success.Printfln("READY: Execute the following command locally to download the dump:")
	pterm.Success.Printfln("")
	pterm.Success.Printfln("          synco receive %s %s", transferSession.Identifier, transferSession.Password)
	pterm.Success.Printfln("")
	pterm.Success.Printfln("When you are finished, stop the server by pressing Ctrl-C.")
	pterm.Success.Printfln("")
	pterm.Success.Printfln("=================================================================================")
	pterm.Success.Printfln("")
}

func (f flowServe) extractDatabaseCredentialsFromFlow() flowPersistenceBackendOptions {
	pterm.Debug.Println("Finding database credentials")
	output := f.readFlowSettings("Neos.Flow.persistence.backendOptions")
	var flowPersistence flowPersistenceBackendOptions
	err := yaml.Unmarshal([]byte(output), &flowPersistence)
	if err != nil {
		pterm.Fatal.Printfln("could not parse output of ./flow configuration:show: %s. Output was: %s", err, output)
	}
	pterm.Info.Printfln("Extracted Database Host %s, User: %s", flowPersistence.Host, flowPersistence.User)
	return flowPersistence
}

func (f flowServe) extractResourceConfigFromFlow() flowResourceOptions {
	pterm.Debug.Println("Finding database credentials")
	output := f.readFlowSettings("Neos.Flow.persistence.backendOptions")
	var opts flowResourceOptions
	err := yaml.Unmarshal([]byte(output), &opts)
	if err != nil {
		pterm.Fatal.Printfln("could not parse output of ./flow configuration:show: %s. Output was: %s", err, output)
	}
	return opts
}

func (f flowServe) readFlowSettings(path string) string {
	cmd := exec.Command("./flow", "configuration:show", "--type", "Settings", "--path", path)
	output, err := util.RunWrappedCommand(cmd)
	if err != nil {
		pterm.Fatal.Printfln("./flow configuration:show did not succeed: %s", err)
	}
	// remove the first line; as it contains the " Configuration "Settings: Neos.Flow.persistence.backendOptions":" line:
	outputParts := strings.SplitN(output, "\n", 2)
	output = outputParts[1]
	return output
}

func (f flowServe) databaseDump(transferSession *serve.TransferSession, flowPersistence flowPersistenceBackendOptions, ignoreContentOfTables []string) {
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
	err = mysql.CreateDump(flowPersistence.ToDbCredentials(), wc, ignoreContentOfTables)
	if err != nil {
		pterm.Fatal.Printfln("could not create SQL dump: %s", err)
	}
	fileSet.MysqlDump.SizeBytes = wc.Size()
	transferSession.Meta.FileSets = append(transferSession.Meta.FileSets, fileSet)
	err = transferSession.UpdateMetadata()
	if err != nil {
		pterm.Fatal.Printfln("could not update SQL dump metadata: %s", err)
	}

	pterm.Info.Printfln("Stored Database Dump in %s", "dump.sql.enc")
}

func (f flowServe) extractResources(transferSession *serve.TransferSession, persistentResourcesBasePath string, baseUri string) {
	indexFileName := "Resources.index.json.enc"
	resourceFilesIndex := make(dto.PublicFilesIndex)

	totalSizeBytes := uint64(0)
	err := filepath.Walk(persistentResourcesBasePath,
		func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				// skip directories on traversal
				return nil
			}

			realPath, err := filepath.EvalSymlinks(filePath)
			if err != nil {
				return err
			}
			realFileInfo, err := os.Lstat(realPath)
			if err != nil {
				return err
			}

			filePath = strings.TrimPrefix(filePath, persistentResourcesBasePath) + baseUri

			// Flow stores files in /..../<resourceId>/<filename>.jpg; so we extract the resourceId here.
			resourceId := path.Base(path.Dir(filePath))

			totalSizeBytes += uint64(realFileInfo.Size())
			resourceFilesIndex["Resources/"+resourceId[0:1]+"/"+resourceId[1:2]+"/"+resourceId[2:3]+"/"+resourceId[3:4]+"/"+resourceId] = dto.PublicFilesIndexEntry{
				SizeBytes: int64(realFileInfo.Size()),
				MTime:     realFileInfo.ModTime().Unix(),
				PublicUri: "<BASE>/" + filePath,
			}
			return nil
		})
	if err != nil {
		log.Println(err)
	}

	bytes, err := json.Marshal(resourceFilesIndex)
	if err != nil {
		pterm.Fatal.Printfln("could not encode resourceFilesIndex: %s", err)
	}

	err = transferSession.EncryptBytesToFile(indexFileName, bytes)
	if err != nil {
		pterm.Fatal.Printfln("could not encrypt to file: %s", err)
	}

	fileSet := &dto.FileSet{
		Name: "Resources",
		Type: dto.TYPE_PUBLICFILES,
		PublicFiles: &dto.FileSetPublicFiles{
			IndexFileName: indexFileName,
			SizeBytes:     totalSizeBytes,
		},
	}
	transferSession.Meta.FileSets = append(transferSession.Meta.FileSets, fileSet)
	err = transferSession.UpdateMetadata()
	if err != nil {
		pterm.Fatal.Printfln("could not update Resource dump metadata: %s", err)
	}
	pterm.Info.Printfln("Extracted Resource Index")
}

func NewFlowFramework() common.ServeFramework {
	return &flowServe{}
}
