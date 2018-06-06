package jobs

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"

	"encoding/json"

	"github.com/bmeg/golib"
	"github.com/bmeg/arachne/aql"
	"github.com/bmeg/arachne/engine"
	"github.com/bmeg/arachne/gdbi"
)

type LocalManager struct {
	baseDir  string
	db        gdbi.GraphDB
	workDir   string
	workQueue chan *aql.JobStatus
}

func NewLocalServer(path string, workDir string, db gdbi.GraphDB) JobManager {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		os.Mkdir(path, 0700)
	}

	workQueue := make(chan *aql.JobStatus, 10)
	for i := 0; i < 4; i++ {
		q := QueryRunner{workDir: workDir, baseDir: path, db: db}
		go q.Process(workQueue)
	}
	return &LocalManager{baseDir: path, db: db, workDir: workDir, workQueue: workQueue}
}

var idRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func randID() string {
	b := make([]rune, 10)
	for i := range b {
		b[i] = idRunes[rand.Intn(len(idRunes))]
	}
	return string(b)
}

func (man *LocalManager) CreateJob(query *aql.GraphQuery) aql.JobStatus {
	id := randID()
	o := aql.JobStatus{
		Id:        id,
		State:     aql.JobState_QUEUED,
		Graph:     query.Graph,
		Query:     query.Query,
		LineCount: 0,
		FileSize:  0,
	}
	man.workQueue <- &o
	return o
}

func (man *LocalManager) ListJobs() chan aql.JobStatus {
	out := make(chan aql.JobStatus, 100)
	defer close(out)
	return out
}

func (man *LocalManager) QueryJob(ctx context.Context,
	graphName string, jobId string, query []*aql.GraphStatement) chan *aql.QueryResult {

	travelers := make(chan gdbi.Traveler, 1000)
	go func() {
		defer close(travelers)
		filePath := man.baseDir + "/" + jobId + ".stream"
		lines, err := golib.ReadFileLines(filePath)
		if err != nil {
			return
		}
		for line := range lines {
			if len(line) > 0 {
				trav := gdbi.Traveler{}
				json.Unmarshal(line, &trav)
				travelers <- trav
			}
		}
	}()

	out := make(chan *aql.QueryResult, 1000)
	go func() {
		defer close(out)
		graph, err := man.db.Graph(graphName)
		compiler := graph.Compiler()
		log.Printf("Pipeline compile: %s", query)
		pipeline, err := compiler.Compile(query)
		if err != nil {
			log.Printf("Pipeline compile error: %s", err)
			return
		}
		log.Printf("%s", pipeline)
		for trav := range travelers {
			res := engine.Convert(gdbi.VertexData, map[string]gdbi.DataType{}, &trav)
			out <- res
		}
	}()

	return out
}


type QueryRunner struct {
	db      gdbi.GraphDB
	baseDir string
	workDir string
}

func (runner *QueryRunner) Process(in chan *aql.JobStatus) {
	for i := range in {
		out, _ := os.Create(runner.baseDir + "/" + i.Id + ".stream")
		runner.run(i.Graph, i.Query, out)
		out.Close()
	}
}

func (runner *QueryRunner) run(graphName string, query []*aql.GraphStatement, out io.Writer) error {
	log.Printf("Starting query job")
	bufsize := 5000
	graph, err := runner.db.Graph(graphName)
	if err != nil {
		return err
	}
	compiler := graph.Compiler()
	pipeline, err := compiler.Compile(query)
	if err != nil {
		return err
	}
	res := engine.Start(context.Background(), pipeline, runner.workDir, bufsize)
	for row := range res {
		err := store(row, out)
		if err != nil {
			return fmt.Errorf("error sending Traversal result: %v", err)
		}
	}
	return nil
}

func store(trav *gdbi.Traveler, out io.Writer) error {
	b, _ := json.Marshal(trav)
	out.Write(b)
	out.Write([]byte("\n"))
	return nil
}
