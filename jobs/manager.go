package jobs

import (
	"context"
	"github.com/bmeg/arachne/aql"
)

type JobManager interface {
	CreateJob(query *aql.GraphQuery) aql.JobStatus
	ListJobs() chan aql.JobStatus
	QueryJob(ctx context.Context, graph string, jobId string, query []*aql.GraphStatement) chan *aql.QueryResult
}
