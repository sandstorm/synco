package test_e2e

import (
	"io/ioutil"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/orlangure/gnomock"
	"github.com/orlangure/gnomock/preset/postgres"
)
import "github.com/rogpeppe/go-internal/testscript"

const pgQueries = `
	create table t_parent (
		id integer generated always as identity primary key,
		name varchar(255) not null
	);
	create table t_child (
		id integer generated always as identity primary key,
		parent_id integer not null references t_parent(id),
		value text
	);
	insert into t_parent (name) values ('alpha');
	insert into t_parent (name) values ('beta');
	insert into t_child (parent_id, value) values (1, 'child-of-alpha');
	insert into t_child (parent_id, value) values (2, 'child-of-beta');
`

func startPostgresDb(t *testing.T) (string, string) {
	t.Helper()
	p := postgres.Preset(
		postgres.WithVersion("16"),
		postgres.WithUser("admin", "password"),
		postgres.WithDatabase("dummy1"),
		postgres.WithQueries(pgQueries),
	)
	var container *gnomock.Container
	var err error
	if reuseDatabaseContainer {
		container, err = gnomock.Start(p, gnomock.WithDebugMode(), gnomock.WithContainerReuse(), gnomock.WithContainerName("synco-test-postgres"))
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

func TestPostgresFrameworkExportsDatabase(t *testing.T) {
	dbHost, dbPort := startPostgresDb(t)
	testscript.Run(t, testscript.Params{
		Dir: "testdata/postgresframework",
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
