package main

import (
	"bitbucket.org/nfnty_admin/std_pkg/cli"
	"bitbucket.org/nfnty_admin/std_pkg/db/mysql"
	"bytes"
	"fmt"
	"github.com/MouseHatGames/go-mysqldump"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

type Configuration struct {
	SourceMysql mysql.Opts `command:"source_mysql"`
	TargetMysql mysql.Opts `command:"target_mysql"`

	// source options
	ChunkSize   int        `command:"chunk_size,default=0"`

	// target options
	QuerySize   int        `command:"query_size,default=1000000"`
	Verbose     int        `command:"verbose,default=0"`
	WorkerCount int        `command:"worker_count,default=10"` // unused for now
}

var c *Configuration

func main() {
	command := cli.Initialize("DB dumper", &c)
	command.OnRun(func() {
		start := time.Now()
		defer func() {
			logrus.Info(time.Now().Sub(start).String())
		}()

		var wg sync.WaitGroup

		var writerGroup sync.WaitGroup

		pr, pw := io.Pipe()


		wg.Add(1)
		go func() {
			db, err := mysql.NewMysqlClient(&c.SourceMysql)
			if err != nil {
				logrus.Fatal(err)
			}
			dumper := mysqldump.NewDumper(db, pw, c.ChunkSize)
			err = dumper.DumpAllTables(c.SourceMysql.Database, &writerGroup)
			if err != nil {
				logrus.Fatal(err)
			}
			pw.Close()
			wg.Done()
		}()

		db, err := mysql.NewMysqlClient(&c.TargetMysql)

		queryWorker := make(chan string, 100)
		wg.Add(1)
		rq := 1

		runningQueries := 0
		rqMutex := sync.Mutex{}
		locked := false
		rqDone := false
		go func() {
			for q := range queryWorker {

				if c.Verbose > 0 {
					os.Stdout.WriteString(fmt.Sprintf("/** Running Query: %d **/;\n", rq))
				}
				rq++
				_, err := db.Exec(q)
				if err != nil {
					logrus.Fatal(err)
				}

				if c.Verbose > 0 {
					os.Stdout.WriteString(q)
				}
				rqMutex.Lock()
				runningQueries--
				if runningQueries == 0 && locked {
					locked = false
					os.Stdout.WriteString("Caught up with running queries, allowing next read block.\n")
					writerGroup.Done()
				}
				if runningQueries == 0 && rqDone {
					break
				}
				rqMutex.Unlock()
			}
			wg.Done()
		}()

		wg.Add(1)
		sq := 1
		go func() {
			w := newChanWriter()
			flusher := make(chan bool, 1)
			ready := make(chan bool, 1)
			if err != nil {
				logrus.Fatal(err)
			}

			go func() {
				for _ = range flusher {
					data := w.Flush()
					q := fmt.Sprintf(`
/*!40101 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

%s
`, data)

					rqMutex.Lock()
					runningQueries++
					if c.Verbose > 0 {
						os.Stdout.WriteString(fmt.Sprintf("/** Sending Query: %d - %d / 50 **/;\n", sq, runningQueries))
					}
					if runningQueries > 50 && !locked {
						locked = true
						writerGroup.Add(1)
						os.Stdout.WriteString("Preventing next read block to give room to run queries.\n")
					}
					rqMutex.Unlock()
					queryWorker <- q
					if c.Verbose > 0 {
						os.Stdout.WriteString(fmt.Sprintf("/** Send Query: %d **/;\n", sq))
					}
					sq++

					ready<-true
				}
			}()

			err = mysqldump.ConvertToSQL(pr, w, flusher, ready, c.QuerySize, mysqldump.ConvertOptions{
				Tables: []string{},
			})
			if err != nil {
				logrus.Fatalf("failed to convert dump file: %s", err.Error())
			}

			wg.Done()
			rqMutex.Lock()
			rqDone = true
			rqMutex.Unlock()
		}()

		wg.Wait()
		// let the istio proxy know we are done
		http.DefaultClient.Timeout = 1 * time.Second
		http.DefaultClient.Post("http://127.0.0.1:15020/quitquitquit", "text/plain", bytes.NewBufferString(""))
	})

	command.Execute()
}

type chanWriter struct {
	buffer []byte
}

func newChanWriter() *chanWriter {
	return &chanWriter{make([]byte, 0, 100000000)}
}

func (w *chanWriter) Flush() []byte {
	defer func() {
		w.buffer = make([]byte, 0, 100000000)
	}()
	return w.buffer
}

func (w *chanWriter) Write(p []byte) (int, error) {
	w.buffer = append(w.buffer, p...)

	return len(p), nil
}

func (w *chanWriter) Close() error {
	return nil
}
