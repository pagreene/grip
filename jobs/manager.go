package jobs

import (
	"github.com/bmeg/arachne/aql"
)

type JobManager interface {
	CreateJob(query *aql.GraphQuery) aql.JobStatus
	ListJobs() chan aql.JobStatus
}
