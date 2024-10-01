package laravelServe

import (
	"encoding/json"
	"fmt"
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/pkg/common"
	"github.com/sandstorm/synco/pkg/common/commonServe"
	"github.com/sandstorm/synco/pkg/common/dto"
	"github.com/sandstorm/synco/pkg/serve"
	"github.com/sandstorm/synco/pkg/util"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type laravelServe struct {
}

func (l laravelServe) Name() string {
	return "Laravel"
}

func (l laravelServe) Detect() bool {
	if _, err := os.Stat("artisan"); err != nil {
		pterm.Debug.Println("./artisan binary not found, thus no installed Laravel Application")
		return false
	}

	return true
}

type laravelDatabaseOptions struct {
	Default     string `json:"default"`
	Connections map[string]laravelDatabaseConnectionOptions
}

type laravelDatabaseConnectionOptions struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (ldo *laravelDatabaseOptions) ToDbCredentials() *common.DbCredentials {
	pterm.Debug.Printfln("taking DB connection: %s", ldo.Default)

	connection, found := ldo.Connections[ldo.Default]
	if !found {
		pterm.Warning.Printfln("Could not extract DB connection, WILL NOT INCLUDE DB DUMP.")
		return nil
	}
	port := 3306
	if len(connection.Port) != 0 {
		port, _ = strconv.Atoi(connection.Port)
	}
	return &common.DbCredentials{
		Host:     connection.Host,
		Port:     port,
		User:     connection.Username,
		Password: connection.Password,
		DbName:   connection.Database,
	}
}

func (l laravelServe) Serve(transferSession *serve.TransferSession) {
	err := transferSession.WithFrameworkAndWebDirectory(l.Name(), "public")
	if err != nil {
		pterm.Fatal.Printfln("Error writing transferSession: %s", err)
	}

	laravelDatabaseCredentials := extractDatabaseCredentialsFromLaravel()
	commonServe.DatabaseDump(transferSession, laravelDatabaseCredentials.ToDbCredentials(), map[string]string{})
	resourceConfig := extractResourceConfig()

	for id, disk := range resourceConfig.Disks {
		if disk.Driver != "local" {
			pterm.Warning.Printfln("Laravel storage driver %s not supported right now - so NOT transferring %s (path=%s)", disk.Driver, id, disk.Root)
			continue
		}
		if disk.Visibility == "public" {
			pterm.Info.Printfln("Extracting public resources for storage %s (driver=%s, path=%s, baseUri=%s)", id, disk.Driver, disk.Root, disk.Url)
			extractAllResourcesFromFolder(transferSession, "Resources "+id, disk.Root, disk.Url)
		} else {
			pterm.Warning.Printfln("Non-public storage not supported right now - so NOT transferring %s (driver=%s, path=%s)", id, disk.Driver, disk.Root)
		}
	}

	transferSession.Meta.State = dto.STATE_READY
	err = transferSession.UpdateMetadata()
	if err != nil {
		pterm.Fatal.Printfln("could not update state: %s", err)
	}
	pterm.Success.Printfln("")
	pterm.Success.Printfln("=================================================================================")
	pterm.Success.Printfln("")
	pterm.Success.Printfln("The dump does NOT contain:")
	pterm.Success.Printfln("- non-public storage")
	pterm.Success.Printfln("")

	transferSession.RenderConnectCommand()

	pterm.Success.Printfln("")
	pterm.Success.Printfln("=================================================================================")
	pterm.Success.Printfln("")
}

func extractDatabaseCredentialsFromLaravel() laravelDatabaseOptions {
	pterm.Debug.Println("Finding database credentials")
	output := runArtisanTinker("echo json_encode(config('database'))")
	var laravelDb laravelDatabaseOptions
	err := json.Unmarshal([]byte(output), &laravelDb)
	if err != nil {
		pterm.Fatal.Printfln("could not parse output of artisan tinker: %s. Output was: %s", err, output)
	}
	pterm.Info.Printfln("Extracted Database Host %s, User: %s", laravelDb.ToDbCredentials().Host, laravelDb.ToDbCredentials().User)
	return laravelDb
}

type laravelFilesystems struct {
	Default string                 `json:"default"`
	Disks   map[string]laravelDisk `json:"disks"`
	Links   map[string]string      `json:"links"`
}

type laravelDisk struct {
	Driver     string `json:"driver"`
	Root       string `json:"root"`
	Url        string `json:"url"`
	Visibility string `json:"visibility"`
}

func extractResourceConfig() laravelFilesystems {
	pterm.Debug.Println("Finding resource(filesystem) configuration")
	output := runArtisanTinker("echo json_encode(config('filesystems'))")
	var opts laravelFilesystems
	err := json.Unmarshal([]byte(output), &opts)
	if err != nil {
		pterm.Fatal.Printfln("could not parse output of artisan tinker: %s. Output was: %s", err, output)
	}
	return opts
}

func runArtisanTinker(tinkerCommand string) string {
	cmd := commonServe.ExecWithVariousPhpInterpreters(fmt.Sprintf("artisan tinker --execute=\"%s\"", tinkerCommand))
	php := os.Getenv("PHP")
	if php != "" {
		// in case the PHP version is specified via the "$PHP" env var, we take this one.
		cmd = exec.Command(php, "artisan", "tinker", "--execute", tinkerCommand)
	}

	output, _, err := util.RunWrappedCommand(cmd)
	if err != nil {
		pterm.Fatal.Printfln("./artisan tinker --execute=\"%s\": %s", tinkerCommand, err)
	}
	return output
}

func NewLaravel() common.ServeFramework {
	return &laravelServe{}
}

func extractAllResourcesFromFolder(transferSession *serve.TransferSession, name, persistentResourcesBasePath string, baseUri string) {
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

			publicUri, err := url.JoinPath(baseUri, filePath)
			if err != nil {
				return err
			}

			totalSizeBytes += uint64(realFileInfo.Size())
			resourceFilesIndex[persistentResourcesBasePath+filePath] = dto.PublicFilesIndexEntry{
				SizeBytes: int64(realFileInfo.Size()),
				MTime:     realFileInfo.ModTime().Unix(),
				PublicUri: publicUri,
			}
			return nil
		})
	if err != nil {
		log.Println(err)
	}

	commonServe.WriteResourcesIndex(transferSession, name, resourceFilesIndex, totalSizeBytes)
}
