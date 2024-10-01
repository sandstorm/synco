package test_e2e

import (
	"github.com/orlangure/gnomock"
	"github.com/orlangure/gnomock/preset/mariadb"
	"github.com/sandstorm/synco/v2/cmd"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)
import "github.com/rogpeppe/go-internal/testscript"

// setting to TRUE greatly speeds up tests
const reuseDatabaseContainer = true

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"synco": func() int {
			cmd.Execute()
			return 0
		},
	}))
}

const queries = `
	drop table if exists t;
	create table t(a int);
	insert into t (a) values (1);
	insert into t (a) values (2);
`

func startDb(t *testing.T) (string, string) {
	t.Helper()
	p := mariadb.Preset(
		mariadb.WithUser("admin", "password"),
		mariadb.WithDatabase("dummy1"),
		mariadb.WithQueries(queries),
	)
	var container *gnomock.Container
	var err error
	if reuseDatabaseContainer {
		// TO DEBUG: gnomock.WithDebugMode(),
		container, err = gnomock.Start(p, gnomock.WithDebugMode(), gnomock.WithContainerReuse(), gnomock.WithContainerName("synco-test-flow"))
	} else {
		container, err = gnomock.Start(p)
		t.Cleanup(func() {
			_ = gnomock.Stop(container)
		})
	}

	if err != nil {
		panic(err)
	}
	return container.Host, strconv.Itoa(container.DefaultPort())
}

func TestFlowFrameworkExportsDatabase(t *testing.T) {
	dbHost, dbPort := startDb(t)
	//dbHost, dbPort := "", ""
	testscript.Run(t, testscript.Params{
		Dir: "testdata/flowframework",
		Setup: func(env *testscript.Env) error {
			env.Setenv("DB_USER", "admin")
			env.Setenv("DB_PASSWORD", "password")
			env.Setenv("DB_NAME", "dummy1")
			env.Setenv("DB_HOST", dbHost)
			env.Setenv("DB_PORT", dbPort)

			return nil
		},
		TestWork: true,
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"fileContentWithTimeout": func(ts *testscript.TestScript, neg bool, args []string) {
				fileName := args[0]
				expectedContent := args[1]
				maxDuration, err := time.ParseDuration(args[2])
				if err != nil {
					ts.Fatalf("Error parsing duration: %s", err)
				}

				startTime := time.Now()
				for {
					file, err := ioutil.ReadFile(fileName)
					if err == nil && strings.TrimSpace(string(file)) == strings.TrimSpace(expectedContent) {
						ts.Logf("Successful file content comparison")
						// no error and matching file content -> success!
						return
					}

					if time.Since(startTime) > maxDuration {
						ts.Fatalf("Error maxDuration")
						break
					}
					time.Sleep(200 * time.Millisecond)
				}
			},
		},
	})
}
