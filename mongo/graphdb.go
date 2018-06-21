package mongo

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bmeg/arachne/aql"
	"github.com/bmeg/arachne/gdbi"
	"github.com/bmeg/arachne/protoutil"
	"github.com/bmeg/arachne/timestamp"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// Config describes the configuration for the mongodb driver.
type Config struct {
	URL                    string
	DBName                 string
	Username               string
	Password               string
	BatchSize              int
	UseAggregationPipeline bool
}

// GraphDB is the base driver that manages multiple graphs in mongo
type GraphDB struct {
	database string
	conf     Config
	session  *mgo.Session
	ts       *timestamp.Timestamp
}

// NewGraphDB creates a new mongo graph database interface
func NewGraphDB(conf Config) (gdbi.GraphDB, error) {
	log.Printf("Starting Mongo Driver")
	database := strings.ToLower(conf.DBName)
	err := aql.ValidateGraphName(database)
	if err != nil {
		return nil, fmt.Errorf("invalid database name: %v", err)
	}

	ts := timestamp.NewTimestamp()
	dialinfo := &mgo.DialInfo{
		Addrs:    []string{conf.URL},
		Database: conf.DBName,
		Username: conf.Username,
		Password: conf.Password,
		AppName:  "arachne",
	}
	session, err := mgo.DialWithInfo(dialinfo)
	if err != nil {
		return nil, err
	}
	session.SetSocketTimeout(1 * time.Hour)
	session.SetSyncTimeout(1 * time.Minute)

	b, _ := session.BuildInfo()
	if !b.VersionAtLeast(3, 6) {
		session.Close()
		return nil, fmt.Errorf("requires mongo 3.6 or later")
	}
	if conf.BatchSize == 0 {
		conf.BatchSize = 1000
	}
	db := &GraphDB{database: database, conf: conf, session: session, ts: &ts}
	for _, i := range db.ListGraphs() {
		db.ts.Touch(i)
	}
	return db, nil
}

// Close the connection
func (ma *GraphDB) Close() error {
	ma.session.Close()
	ma.session = nil
	return nil
}

// VertexCollection returns a *mgo.Collection
func (ma *GraphDB) VertexCollection(session *mgo.Session, graph string) *mgo.Collection {
	return session.DB(ma.database).C(fmt.Sprintf("%s_vertices", graph))
}

// EdgeCollection returns a *mgo.Collection
func (ma *GraphDB) EdgeCollection(session *mgo.Session, graph string) *mgo.Collection {
	return session.DB(ma.database).C(fmt.Sprintf("%s_edges", graph))
}

// AddGraph creates a new graph named `graph`
func (ma *GraphDB) AddGraph(graph string) error {
	err := aql.ValidateGraphName(graph)
	if err != nil {
		return err
	}

	session := ma.session.Copy()
	session.ResetIndexCache()
	defer session.Close()
	defer ma.ts.Touch(graph)

	graphs := session.DB(ma.database).C("graphs")
	err = graphs.Insert(bson.M{"_id": graph})
	if err != nil {
		return fmt.Errorf("failed to insert graph %s: %v", graph, err)
	}

	e := ma.EdgeCollection(session, graph)
	err = e.EnsureIndex(mgo.Index{
		Key:        []string{"$hashed:from"},
		Unique:     false,
		DropDups:   false,
		Sparse:     false,
		Background: true,
	})
	if err != nil {
		return fmt.Errorf("failed create index for graph %s: %v", graph, err)
	}
	err = e.EnsureIndex(mgo.Index{
		Key:        []string{"$hashed:to"},
		Unique:     false,
		DropDups:   false,
		Sparse:     false,
		Background: true,
	})
	if err != nil {
		return fmt.Errorf("failed create index for graph %s: %v", graph, err)
	}
	err = e.EnsureIndex(mgo.Index{
		Key:        []string{"$hashed:label"},
		Unique:     false,
		DropDups:   false,
		Sparse:     false,
		Background: true,
	})
	if err != nil {
		return fmt.Errorf("failed create index for graph %s: %v", graph, err)
	}

	v := ma.VertexCollection(session, graph)
	err = v.EnsureIndex(mgo.Index{
		Key:        []string{"$hashed:label"},
		Unique:     false,
		DropDups:   false,
		Sparse:     false,
		Background: true,
	})
	if err != nil {
		return fmt.Errorf("failed create index for graph %s: %v", graph, err)
	}

	return nil
}

// DeleteGraph deletes `graph`
func (ma *GraphDB) DeleteGraph(graph string) error {
	session := ma.session.Copy()
	defer session.Close()
	defer ma.ts.Touch(graph)

	g := session.DB(ma.database).C("graphs")
	v := ma.VertexCollection(session, graph)
	e := ma.EdgeCollection(session, graph)

	verr := v.DropCollection()
	if verr != nil {
		log.Printf("Drop vertex collection failed: %v", verr)
	}
	eerr := e.DropCollection()
	if eerr != nil {
		log.Printf("Drop edge collection failed: %v", eerr)
	}
	gerr := g.RemoveId(graph)
	if gerr != nil {
		log.Printf("Remove graph id failed: %v", gerr)
	}

	if verr != nil || eerr != nil || gerr != nil {
		return fmt.Errorf("failed to delete graph: %s; %s; %s", verr, eerr, gerr)
	}

	return nil
}

// ListGraphs lists the graphs managed by this driver
func (ma *GraphDB) ListGraphs() []string {
	session := ma.session.Copy()
	defer session.Close()

	out := make([]string, 0, 100)
	g := session.DB(ma.database).C("graphs")

	iter := g.Find(nil).Iter()
	defer iter.Close()
	if err := iter.Err(); err != nil {
		log.Println("ListGraphs error:", err)
	}
	result := map[string]interface{}{}
	for iter.Next(&result) {
		out = append(out, result["_id"].(string))
	}
	if err := iter.Err(); err != nil {
		log.Println("ListGraphs error:", err)
	}

	return out
}

// Graph obtains the gdbi.DBI for a particular graph
func (ma *GraphDB) Graph(graph string) (gdbi.GraphInterface, error) {
	found := false
	for _, gname := range ma.ListGraphs() {
		if graph == gname {
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("graph '%s' was not found", graph)
	}
	return &Graph{
		ar:        ma,
		ts:        ma.ts,
		graph:     graph,
		batchSize: ma.conf.BatchSize,
	}, nil
}

// GetSchema returns the schema of a specific graph in the database
func (ma *GraphDB) GetSchema(graph string, sampleN int) (*aql.GraphSchema, error) {
	vSchema, err := ma.getVertexSchema(graph, sampleN)
	if err != nil {
		return nil, fmt.Errorf("getting vertex schema: %v", err)
	}
	eSchema, err := ma.getEdgeSchema(graph, sampleN)
	if err != nil {
		return nil, fmt.Errorf("getting edge schema: %v", err)
	}
	schema := &aql.GraphSchema{Vertices: vSchema, Edges: eSchema}
	log.Printf("Graph schema: %+v", schema)
	return schema, nil
}

func (ma *GraphDB) getVertexSchema(graph string, n int) ([]*aql.Vertex, error) {
	out := []*aql.Vertex{}

	session := ma.session.Copy()
	defer session.Close()
	pipe := []bson.M{
		{
			"$group": bson.M{
				"_id":  "$label",
				"data": bson.M{"$push": "$data"},
			},
		},
		{
			"$project": bson.M{
				"data": bson.M{
					"$slice": []interface{}{"$data", n},
				},
			},
		},
	}

	v := ma.VertexCollection(session, graph)
	iter := v.Pipe(pipe).Iter()
	result := &schema{}
	for iter.Next(result) {
		schema := make(map[string]interface{})
		for i, v := range result.Data {
			out := GetDataFieldTypes(v)
			MergeMaps(schema, out)
			result.Data[i] = out
		}
		vs := &aql.Vertex{Label: result.Label, Data: protoutil.AsStruct(schema)}
		out = append(out, vs)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func resolveLabel(col *mgo.Collection, ids []string) (string, error) {
	pipe := []bson.M{
		{
			"$match": bson.M{
				"_id": bson.M{"$in": ids},
			},
		},
		{
			"$project": bson.M{
				"_id":   -1,
				"label": 1,
			},
		},
	}
	iter := col.Pipe(pipe).Iter()
	var last string
	result := map[string]string{}
	for iter.Next(&result) {
		if last != "" && last != result["label"] {
			return "", fmt.Errorf("resolved to multiple labels: %s %s", last, result)
		}
		last = result["label"]
	}
	return last, nil
}

func (ma *GraphDB) getEdgeSchema(graph string, n int) ([]*aql.Edge, error) {
	out := []*aql.Edge{}

	session := ma.session.Copy()
	defer session.Close()
	pipe := []bson.M{
		{
			"$group": bson.M{
				"_id":  "$label",
				"from": bson.M{"$push": "$from"},
				"to":   bson.M{"$push": "$to"},
				"data": bson.M{"$push": "$data"},
			},
		},
		{
			"$project": bson.M{
				"_id": 1,
				"to": bson.M{
					"$slice": []interface{}{"$to", n},
				},
				"from": bson.M{
					"$slice": []interface{}{"$from", n},
				},
				"data": bson.M{
					"$slice": []interface{}{"$data", n},
				},
			},
		},
	}

	e := ma.EdgeCollection(session, graph)
	iter := e.Pipe(pipe).Iter()
	result := &schema{}
	for iter.Next(result) {
		schema := make(map[string]interface{})
		for i, v := range result.Data {
			out := GetDataFieldTypes(v)
			MergeMaps(schema, out)
			result.Data[i] = out
		}
		from, err := resolveLabel(ma.VertexCollection(session, graph), result.From)
		if err != nil {
			return nil, err
		}
		to, err := resolveLabel(ma.VertexCollection(session, graph), result.To)
		if err != nil {
			return nil, err
		}
		es := &aql.Edge{Label: result.Label, From: from, To: to, Data: protoutil.AsStruct(schema)}
		out = append(out, es)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

type schema struct {
	Label string                   `bson:"_id"`
	From  []string                 `bson:"from"`
	To    []string                 `bson:"to"`
	Data  []map[string]interface{} `bson:"data"`
}

// MergeMaps deeply merges two maps
func MergeMaps(x1, x2 interface{}) interface{} {
	switch x1 := x1.(type) {
	case map[string]interface{}:
		x2, ok := x2.(map[string]interface{})
		if !ok {
			return x1
		}
		for k, v2 := range x2 {
			if v1, ok := x1[k]; ok {
				x1[k] = MergeMaps(v1, v2)
			} else {
				x1[k] = v2
			}
		}
	case nil:
		x2, ok := x2.(map[string]interface{})
		if ok {
			return x2
		}
	}
	return x1
}

// GetDataFieldTypes iterates over the data map and determines the type of each field
func GetDataFieldTypes(data map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for key, val := range data {
		if vMap, ok := val.(map[string]interface{}); ok {
			out[key] = GetDataFieldTypes(vMap)
			continue
		}
		if vSlice, ok := val.([]interface{}); ok {
			var vType interface{}
			vType = []interface{}{aql.FieldType_UNKNOWN.String()}
			if len(vSlice) > 0 {
				vSliceVal := vSlice[0]
				if vSliceValMap, ok := vSliceVal.(map[string]interface{}); ok {
					vType = []map[string]interface{}{GetDataFieldTypes(vSliceValMap)}
				} else {
					vType = []interface{}{GetFieldType(vSliceVal)}
				}
			}
			out[key] = vType
			continue
		}
		out[key] = GetFieldType(val)
	}
	return out
}

// GetFieldType returns the aql.FieldType for a value
func GetFieldType(field interface{}) string {
	switch field.(type) {
	case string:
		return aql.FieldType_STRING.String()
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return aql.FieldType_NUMERIC.String()
	case float32, float64:
		return aql.FieldType_NUMERIC.String()
	case bool:
		return aql.FieldType_BOOL.String()
	default:
		return aql.FieldType_UNKNOWN.String()
	}
}
