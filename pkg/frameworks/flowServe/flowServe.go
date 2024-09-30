package flowServe

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/pkg/common"
	"github.com/sandstorm/synco/pkg/common/dto"
	"github.com/sandstorm/synco/pkg/serve"
	"github.com/sandstorm/synco/pkg/util"
	"github.com/sandstorm/synco/pkg/util/mysql"
	"gopkg.in/yaml.v3"
	"log"
	"net/url"
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
			pterm.Info.Printfln("target '%s' is configured as follows:", persistentTargetName)
			pterm.Info.Printfln("  '%s'", target)
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
	whereClauseForTables := map[string]string{
		// event log can be HUGE and is usually not needed.
		"neos_neos_eventlog_domain_model_event": "FALSE",
		// thumbnails can be regenerated
		"neos_media_domain_model_thumbnail": "FALSE",

		// skip persistent resources which are purely for thumbnails
		"neos_flow_resourcemanagement_persistentresource": `
			NOT EXISTS (
    			SELECT 1
				FROM neos_media_domain_model_thumbnail th
				WHERE
					th.resource IS NOT NULL
					AND th.resource = neos_flow_resourcemanagement_persistentresource.persistence_object_identifier
				)`,
	}
	if transferSession.DumpAll {
		whereClauseForTables = map[string]string{}
	}
	db := f.databaseDump(transferSession, flowPersistence, whereClauseForTables)
	flowResourceConfig := f.extractResourceConfigFromFlow()
	persistentTarget := flowResourceConfig.FindPersistentTarget()

	if persistentTarget == nil {
		pterm.Warning.Printfln("falling back to extracting locations from default location.")
		// fallback to extracting resources from default location
		f.extractAllResourcesFromFolder(transferSession, "./Web/_Resources/Persistent", "_Resources/Persistent")
	} else if persistentTarget.IsS3Target() {
		pterm.Info.Printfln("Extracting resources for S3Target (baseUri=%s)", persistentTarget.TargetOptions.BaseUri)
		f.extractResourcesFromS3(transferSession, db, persistentTarget, whereClauseForTables)
	} else if persistentTarget.IsFileSystemTarget() {
		if transferSession.DumpAll {
			pterm.Info.Printfln("Extracting ALL resources for FileSystemTarget (path=%s, baseUri=%s)", persistentTarget.TargetOptions.Path, persistentTarget.TargetOptions.BaseUri)
			f.extractAllResourcesFromFolder(transferSession, persistentTarget.TargetOptions.Path, persistentTarget.TargetOptions.BaseUri)
		} else {
			pterm.Info.Printfln("Extracting resources (but skipping thumbnails) for FileSystemTarget (path=%s, baseUri=%s)", persistentTarget.TargetOptions.Path, persistentTarget.TargetOptions.BaseUri)
			f.extractResourcesFromFolderSkippingThumbnails(transferSession, db, persistentTarget, whereClauseForTables)
		}
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
	if !transferSession.DumpAll {
		pterm.Success.Printfln("The dump does NOT contain:")
		pterm.Success.Printfln("- neos_neos_eventlog_domain_model_event (usually huge and not needed)")
		pterm.Success.Printfln("- neos_media_domain_model_thumbnail (can be regenerated on the client)")
		pterm.Success.Printfln("- partially neos_flow_resourcemanagement_persistentresource (thumbnails not included)")
		pterm.Success.Printfln("")
		pterm.Success.Printfln("In case you want to dump all tables, run with --all.")
	}

	pterm.Success.Printfln("READY: Execute the following command on the target system to download the dump:")
	pterm.Success.Printfln("")
	pterm.Success.Printfln("          # locally: ")
	pterm.Success.Printfln("          synco receive %s %s", transferSession.Identifier, transferSession.Password)
	pterm.Success.Printfln("")
	pterm.Success.Printfln("          # on another server:")
	pterm.Success.Printfln("          curl https://sandstorm.github.io/synco/synco | sh -s - receive %s %s", transferSession.Identifier, transferSession.Password)
	pterm.Success.Printfln("")

	if !transferSession.KeepFiles {
		pterm.Success.Printfln("When you are finished, stop the server by pressing Ctrl-C")
		pterm.Success.Printfln("to have synco clean up your files.")
	} else {
		pterm.Success.Printfln("You are finished.")
		pterm.Warning.Printfln("Syno will --keep the file '%s'.", *transferSession.WorkDir)
		pterm.Warning.Printfln("You will have to remove it manually!!!")
	}

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
	pterm.Debug.Println("Finding resource configuration")
	output := f.readFlowSettings("Neos.Flow.resource")
	var opts flowResourceOptions
	err := yaml.Unmarshal([]byte(output), &opts)
	if err != nil {
		pterm.Fatal.Printfln("could not parse output of ./flow configuration:show: %s. Output was: %s", err, output)
	}
	return opts
}

func (f flowServe) readFlowSettings(path string) string {
	// try to auto-detect PHP versions by trying different PHP interpreters
	cmd := exec.Command("sh", "-c", fmt.Sprintf("./flow configuration:show --type Settings --path %s || php82 flow configuration:show --type Settings --path %s || php81 flow configuration:show --type Settings --path %s || php80 flow configuration:show --type Settings --path %s || php74 flow configuration:show --type Settings --path %s || php8.2 flow configuration:show --type Settings --path %s || php8.1 flow configuration:show --type Settings --path %s || php8.0 flow configuration:show --type Settings --path %s || php7.4 flow configuration:show --type Settings --path %s", path, path, path, path, path, path, path, path, path))
	php := os.Getenv("PHP")
	if php != "" {
		// in case the PHP version is specified via the "$PHP" env var, we take this one.
		cmd = exec.Command(php, "flow", "configuration:show", "--type", "Settings", "--path", path)
	}

	output, _, err := util.RunWrappedCommand(cmd)
	if err != nil {
		pterm.Fatal.Printfln("./flow configuration:show did not succeed: %s", err)
	}
	// remove the first line; as it contains the " Configuration "Settings: Neos.Flow.persistence.backendOptions":" line:
	outputParts := strings.SplitN(output, "\n", 2)
	output = outputParts[1]
	return output
}

func (f flowServe) databaseDump(transferSession *serve.TransferSession, flowPersistence flowPersistenceBackendOptions, whereClauseForTables map[string]string) *sql.DB {
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
	db, err := mysql.CreateDump(flowPersistence.ToDbCredentials(), wc, whereClauseForTables)
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
	return db
}

func (f flowServe) extractAllResourcesFromFolder(transferSession *serve.TransferSession, persistentResourcesBasePath string, baseUri string) {
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
				pterm.Error.Printfln("Could NOT evaluate symlinks (skipping): %s: %s", filePath, err)
				return nil
			}
			realFileInfo, err := os.Lstat(realPath)
			if err != nil {
				pterm.Error.Printfln("Could NOT read file info (skipping): %s: %s", realPath, err)
				return nil
			}

			filePath = strings.TrimPrefix(filePath, persistentResourcesBasePath)

			// Flow stores files in /..../<resourceSha1>/<filename>.jpg; so we extract the resourceSha1 here.
			resourceSha1 := path.Base(path.Dir(filePath))

			publicUri, err := url.JoinPath(baseUri, filePath)
			if err != nil {
				return err
			}

			totalSizeBytes += uint64(realFileInfo.Size())
			resourceFilesIndex["Resources/"+resourceSha1[0:1]+"/"+resourceSha1[1:2]+"/"+resourceSha1[2:3]+"/"+resourceSha1[3:4]+"/"+resourceSha1] = dto.PublicFilesIndexEntry{
				SizeBytes: int64(realFileInfo.Size()),
				MTime:     realFileInfo.ModTime().Unix(),
				PublicUri: "<BASE>/" + publicUri,
			}
			return nil
		})
	if err != nil {
		log.Println(err)
	}

	f.writeResourcesIndex(transferSession, resourceFilesIndex, totalSizeBytes)
}

func (f flowServe) extractResourcesFromS3(transferSession *serve.TransferSession, db *sql.DB, persistentTarget *flowResourceTarget, whereClauseForTables map[string]string) {
	extraWhereClause := "true"
	if len(whereClauseForTables["neos_flow_resourcemanagement_persistentresource"]) > 0 {
		extraWhereClause = whereClauseForTables["neos_flow_resourcemanagement_persistentresource"]
	}
	resourceFilesIndex := make(dto.PublicFilesIndex)
	totalSizeBytes := uint64(0)
	q := fmt.Sprintf(`
		SELECT
			sha1, filename, filesize
		FROM
			neos_flow_resourcemanagement_persistentresource
		WHERE collectionname = 'persistent' AND %s`, extraWhereClause)

	rows, err := db.Query(q)
	if err != nil {
		pterm.Fatal.Printfln("could query for resources: %s", err)
	}
	defer rows.Close()

	var resourceSha1, filename string
	var filesize uint64
	for rows.Next() {
		err := rows.Scan(&resourceSha1, &filename, &filesize)
		if err != nil {
			pterm.Fatal.Printfln("error loading DB row: %s", err)
		}

		totalSizeBytes += filesize
		escapedFileName := url.PathEscape(filename)
		// HACK: this is how it works for Neos / Flow. Probably not all escapes done
		escapedFileName = strings.ReplaceAll(escapedFileName, "+", "%2B")
		resourceFilesIndex["Resources/"+resourceSha1[0:1]+"/"+resourceSha1[1:2]+"/"+resourceSha1[2:3]+"/"+resourceSha1[3:4]+"/"+resourceSha1] = dto.PublicFilesIndexEntry{
			SizeBytes: int64(filesize),
			MTime:     0,
			PublicUri: persistentTarget.TargetOptions.BaseUri + resourceSha1 + "/" + escapedFileName,
		}
	}
	err = rows.Err()
	if err != nil {
		pterm.Fatal.Printfln("error iterating rows: %s", err)
	}

	f.writeResourcesIndex(transferSession, resourceFilesIndex, totalSizeBytes)
}

func (f flowServe) writeResourcesIndex(transferSession *serve.TransferSession, resourceFilesIndex dto.PublicFilesIndex, totalSizeBytes uint64) {
	indexFileName := "Resources.index.json.enc"

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

func (f flowServe) extractResourcesFromFolderSkippingThumbnails(transferSession *serve.TransferSession, db *sql.DB, persistentTarget *flowResourceTarget, whereClauseForTables map[string]string) {
	extraWhereClause := "true"
	if len(whereClauseForTables["neos_flow_resourcemanagement_persistentresource"]) > 0 {
		extraWhereClause = whereClauseForTables["neos_flow_resourcemanagement_persistentresource"]
	}
	resourceFilesIndex := make(dto.PublicFilesIndex)
	totalSizeBytes := uint64(0)
	q := fmt.Sprintf(`
		SELECT
			sha1, filename, filesize
		FROM
			neos_flow_resourcemanagement_persistentresource
		WHERE collectionname = 'persistent' AND %s`, extraWhereClause)

	rows, err := db.Query(q)
	if err != nil {
		pterm.Fatal.Printfln("could query for resources: %s", err)
	}
	defer rows.Close()

	var resourceSha1, filename string
	var filesize uint64
	for rows.Next() {
		err := rows.Scan(&resourceSha1, &filename, &filesize)
		if err != nil {
			pterm.Fatal.Printfln("error loading DB row: %s", err)
		}

		totalSizeBytes += filesize
		escapedFileName := url.PathEscape(filename)
		// HACK: this is how it works for Neos / Flow. Probably not all escapes done
		escapedFileName = strings.ReplaceAll(escapedFileName, "+", "%2B")
		resourceFilesIndex["Resources/"+resourceSha1[0:1]+"/"+resourceSha1[1:2]+"/"+resourceSha1[2:3]+"/"+resourceSha1[3:4]+"/"+resourceSha1] = dto.PublicFilesIndexEntry{
			SizeBytes: int64(filesize),
			MTime:     0,
			PublicUri: "<BASE>/" + persistentTarget.TargetOptions.BaseUri + resourceSha1[0:1] + "/" + resourceSha1[1:2] + "/" + resourceSha1[2:3] + "/" + resourceSha1[3:4] + "/" + resourceSha1 + "/" + escapedFileName,
		}
	}
	err = rows.Err()
	if err != nil {
		pterm.Fatal.Printfln("error iterating rows: %s", err)
	}

	f.writeResourcesIndex(transferSession, resourceFilesIndex, totalSizeBytes)
}

func NewFlowFramework() common.ServeFramework {
	return &flowServe{}
}
