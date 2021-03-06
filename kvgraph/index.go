package kvgraph

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/bmeg/grip/gripql"
	"github.com/bmeg/grip/jsonpath"
	"github.com/bmeg/grip/protoutil"
	structpb "github.com/golang/protobuf/ptypes/struct"
	log "github.com/sirupsen/logrus"
	"github.com/spenczar/tdigest"
)

func (kgraph *KVGraph) setupGraphIndex(graph string) error {
	return kgraph.idx.AddField(fmt.Sprintf("%s.label", graph))
}

func (kgraph *KVGraph) deleteGraphIndex(graph string) {
	fields := kgraph.idx.ListFields()
	for _, f := range fields {
		t := strings.Split(f, ".")
		if t[0] == graph {
			kgraph.idx.RemoveField(f)
		}
	}
}

func normalizePath(path string) string {
	path = jsonpath.GetJSONPath(path)
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "data.")
	return path
}

func vertexIdxStruct(v *gripql.Vertex) map[string]interface{} {
	k := map[string]interface{}{
		"label": v.Label,
		"v":     map[string]interface{}{v.Label: protoutil.AsMap(v.Data)},
	}
	return k
}

//AddVertexIndex add index to vertices
func (kgdb *KVInterfaceGDB) AddVertexIndex(label string, field string) error {
	log.WithFields(log.Fields{"label": label, "field": field}).Info("Adding vertex index")
	field = normalizePath(field)
	//TODO kick off background process to reindex existing data
	return kgdb.kvg.idx.AddField(fmt.Sprintf("%s.v.%s.%s", kgdb.graph, label, field))
}

//DeleteVertexIndex delete index from vertices
func (kgdb *KVInterfaceGDB) DeleteVertexIndex(label string, field string) error {
	log.WithFields(log.Fields{"label": label, "field": field}).Info("Deleting vertex index")
	field = normalizePath(field)
	return kgdb.kvg.idx.RemoveField(fmt.Sprintf("%s.v.%s.%s", kgdb.graph, label, field))
}

//GetVertexIndexList lists out all the vertex indices for a graph
func (kgdb *KVInterfaceGDB) GetVertexIndexList() chan gripql.IndexID {
	log.Debug("Running GetVertexIndexList")
	out := make(chan gripql.IndexID)
	go func() {
		defer close(out)
		fields := kgdb.kvg.idx.ListFields()
		for _, f := range fields {
			t := strings.Split(f, ".")
			if len(t) > 3 {
				a := gripql.IndexID{Graph: kgdb.graph, Label: t[2], Field: t[3]}
				out <- a
			}
		}
	}()
	return out
}

//VertexLabelScan produces a channel of all vertex ids in a graph
//that match a given label
func (kgdb *KVInterfaceGDB) VertexLabelScan(ctx context.Context, label string) chan string {
	log.WithFields(log.Fields{"label": label}).Debug("Running VertexLabelScan")
	//TODO: Make this work better
	out := make(chan string, 100)
	go func() {
		defer close(out)
		//log.Printf("Searching %s %s", fmt.Sprintf("%s.label", kgdb.graph), label)
		for i := range kgdb.kvg.idx.GetTermMatch(ctx, fmt.Sprintf("%s.label", kgdb.graph), label) {
			//log.Printf("Found: %s", i)
			out <- i
		}
	}()
	return out
}

//GetVertexTermAggregation get count of every term across vertices
func (kgdb *KVInterfaceGDB) GetVertexTermAggregation(ctx context.Context, label string, field string, size uint32) (*gripql.AggregationResult, error) {
	log.WithFields(log.Fields{"label": label, "field": field, "size": size}).Debug("Running GetVertexTermAggregation")
	out := &gripql.AggregationResult{
		Buckets: []*gripql.AggregationResultBucket{},
	}

	namespace := jsonpath.GetNamespace(field)
	if namespace != jsonpath.Current {
		return nil, fmt.Errorf("invalid field path")
	}
	field = normalizePath(field)

	for tcount := range kgdb.kvg.idx.FieldTermCounts(fmt.Sprintf("%s.v.%s.%s", kgdb.graph, label, field)) {
		var t *structpb.Value
		if tcount.String != "" {
			t = protoutil.WrapValue(tcount.String)
		} else {
			t = protoutil.WrapValue(tcount.Number)
		}
		out.SortedInsert(&gripql.AggregationResultBucket{Key: t, Value: float64(tcount.Count)})
		if size > 0 {
			if len(out.Buckets) > int(size) {
				out.Buckets = out.Buckets[:size]
			}
		}
	}

	return out, nil
}

//GetVertexHistogramAggregation get binned counts of a term across vertices
func (kgdb *KVInterfaceGDB) GetVertexHistogramAggregation(ctx context.Context, label string, field string, interval uint32) (*gripql.AggregationResult, error) {
	log.WithFields(log.Fields{"label": label, "field": field, "interval": interval}).Debug("Running GetVertexHistogramAggregation")
	out := &gripql.AggregationResult{
		Buckets: []*gripql.AggregationResultBucket{},
	}

	namespace := jsonpath.GetNamespace(field)
	if namespace != jsonpath.Current {
		return nil, fmt.Errorf("invalid field path")
	}
	field = normalizePath(field)

	min := kgdb.kvg.idx.FieldTermNumberMin(fmt.Sprintf("%s.v.%s.%s", kgdb.graph, label, field))
	max := kgdb.kvg.idx.FieldTermNumberMax(fmt.Sprintf("%s.v.%s.%s", kgdb.graph, label, field))

	i := float64(interval)
	for bucket := math.Floor(min/i) * i; bucket <= max; bucket += i {
		var count uint64
		for tcount := range kgdb.kvg.idx.FieldTermNumberRange(fmt.Sprintf("%s.v.%s.%s", kgdb.graph, label, field), bucket, bucket+i) {
			count += tcount.Count
		}
		out.Buckets = append(out.Buckets, &gripql.AggregationResultBucket{Key: protoutil.WrapValue(bucket), Value: float64(count)})
	}

	return out, nil
}

//GetVertexPercentileAggregation get percentiles of a term across vertices
func (kgdb *KVInterfaceGDB) GetVertexPercentileAggregation(ctx context.Context, label string, field string, percents []float64) (*gripql.AggregationResult, error) {
	log.WithFields(log.Fields{"label": label, "field": field, "percents": percents}).Debug("Running GetVertexPercentileAggregation")
	out := &gripql.AggregationResult{
		Buckets: []*gripql.AggregationResultBucket{},
	}

	namespace := jsonpath.GetNamespace(field)
	if namespace != jsonpath.Current {
		return nil, fmt.Errorf("invalid field path")
	}
	field = normalizePath(field)

	td := tdigest.New()
	for val := range kgdb.kvg.idx.FieldNumbers(fmt.Sprintf("%s.v.%s.%s", kgdb.graph, label, field)) {
		td.Add(val, 1)
	}
	for _, p := range percents {
		q := td.Quantile(p / 100)
		out.Buckets = append(out.Buckets, &gripql.AggregationResultBucket{Key: protoutil.WrapValue(p), Value: q})
	}

	return out, nil
}
