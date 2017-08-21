package mongo

import (
	"context"
	"fmt"
	"github.com/bmeg/arachne/aql"
	"github.com/bmeg/arachne/gdbi"
	"gopkg.in/mgo.v2"
	"log"
)

func NewMongoArachne(url string, database string) gdbi.ArachneInterface {
	session, err := mgo.Dial(url)
	if err != nil {
		log.Printf("%s", err)
	}
	db := session.DB(database)
	return &MongoArachne{db}
}

type MongoArachne struct {
	db *mgo.Database
}

type MongoGraph struct {
	vertices *mgo.Collection
	edges    *mgo.Collection
}

func (self *MongoArachne) AddGraph(graph string) error {
	graphs := self.db.C(fmt.Sprintf("graphs"))
	graphs.Insert(map[string]string{"_id": graph})
	return nil
}

func (self *MongoArachne) Close() {
	self.db.Logout()
}

func (self *MongoArachne) DeleteGraph(graph string) error {
	g := self.db.C(fmt.Sprintf("graphs"))
	v := self.db.C(fmt.Sprintf("%s_vertices", graph))
	e := self.db.C(fmt.Sprintf("%s_edges", graph))
	v.DropCollection()
	e.DropCollection()
	g.RemoveId(graph)
	return nil
}

func (self *MongoArachne) GetGraphs() []string {
	out := make([]string, 0, 100)
	return out
}

func (self *MongoArachne) Graph(graph string) gdbi.DBI {
	return &MongoGraph{
		self.db.C(fmt.Sprintf("%s_vertices", graph)),
		self.db.C(fmt.Sprintf("%s_edges", graph)),
	}
}

func (self *MongoArachne) Query(graph string) gdbi.QueryInterface {
	return self.Graph(graph).Query()
}

func (self *MongoGraph) Query() gdbi.QueryInterface {
	return gdbi.NewPipeEngine(self, false)
}

func (self *MongoGraph) GetEdge(id string, loadProp bool) *aql.Edge {
	d := map[string]interface{}{}
	q := self.vertices.FindId(id)
	q.One(d)
	v := UnpackEdge(d)
	return &v
}

func (self *MongoGraph) GetVertex(key string, load bool) *aql.Vertex {
	d := map[string]interface{}{}
	q := self.vertices.FindId(key)
	q.One(d)
	v := UnpackVertex(d)
	return &v
}

func (self *MongoGraph) SetVertex(vertex aql.Vertex) error {
	_, err := self.vertices.UpsertId(vertex.Gid, PackVertex(vertex))
	return err
}

func (self *MongoGraph) SetEdge(edge aql.Edge) error {
	if edge.Gid != "" {
		_, err := self.edges.UpsertId(edge.Gid, PackEdge(edge))
		return err
	}
	err := self.edges.Insert(PackEdge(edge))
	return err
}

func (self *MongoGraph) DelVertex(key string) error {
	return self.vertices.RemoveId(key)
}

func (self *MongoGraph) DelEdge(key string) error {
	return self.edges.RemoveId(key)
}

func (self *MongoGraph) GetVertexList(ctx context.Context, load bool) chan aql.Vertex {
	o := make(chan aql.Vertex, 100)
	go func() {
		defer close(o)
		iter := self.vertices.Find(nil).Iter()
		result := map[string]interface{}{}
		for iter.Next(&result) {
			v := UnpackVertex(result)
			o <- v
		}
		if err := iter.Close(); err != nil {
			//do something
		}
	}()
	return o
}

func (self *MongoGraph) GetEdgeList(ctx context.Context, loadProp bool) chan aql.Edge {
	o := make(chan aql.Edge, 100)
	go func() {
		defer close(o)
		iter := self.edges.Find(nil).Iter()
		result := map[string]interface{}{}
		for iter.Next(&result) {
			v := UnpackEdge(result)
			o <- v
		}
		if err := iter.Close(); err != nil {
			//do something
		}
	}()
	return o
}

func (self *MongoGraph) GetOutList(ctx context.Context, key string, load bool, filter gdbi.EdgeFilter) chan aql.Vertex {
	o := make(chan aql.Vertex, 100)
	go func() {
		defer close(o)
		selection := map[string]interface{}{
			FIELD_SRC: key,
		}
		iter := self.edges.Find(selection).Iter()
		result := map[string]interface{}{}
		for iter.Next(&result) {
			send := false
			if filter != nil {
				e := UnpackEdge(result)
				if filter(e) {
					send = true
				}
			} else {
				send = true
			}
			if send {
				q := self.vertices.FindId(result[FIELD_DST])
				d := map[string]interface{}{}
				q.One(d)
				v := UnpackVertex(d)
				o <- v
			}
		}
	}()
	return o
}

func (self *MongoGraph) GetInList(ctx context.Context, key string, load bool, filter gdbi.EdgeFilter) chan aql.Vertex {
	o := make(chan aql.Vertex, 100)
	go func() {
		defer close(o)
		selection := map[string]interface{}{
			FIELD_DST: key,
		}
		iter := self.edges.Find(selection).Iter()
		result := map[string]interface{}{}
		for iter.Next(&result) {
			send := false
			if filter != nil {
				e := UnpackEdge(result)
				if filter(e) {
					send = true
				}
			} else {
				send = true
			}
			if send {
				q := self.vertices.FindId(result[FIELD_SRC])
				d := map[string]interface{}{}
				q.One(d)
				v := UnpackVertex(d)
				o <- v
			}
		}
	}()
	return o
}

func (self *MongoGraph) GetOutEdgeList(ctx context.Context, key string, load bool, filter gdbi.EdgeFilter) chan aql.Edge {
	o := make(chan aql.Edge, 100)
	go func() {
		defer close(o)
		selection := map[string]interface{}{
			FIELD_SRC: key,
		}
		iter := self.edges.Find(selection).Iter()
		result := map[string]interface{}{}
		for iter.Next(&result) {
			send := false
			e := UnpackEdge(result)
			if filter != nil {
				if filter(e) {
					send = true
				}
			} else {
				send = true
			}
			if send {
				o <- e
			}
		}
	}()
	return o
}

func (self *MongoGraph) GetInEdgeList(ctx context.Context, key string, load bool, filter gdbi.EdgeFilter) chan aql.Edge {
	o := make(chan aql.Edge, 100)
	go func() {
		defer close(o)
		selection := map[string]interface{}{
			FIELD_DST: key,
		}
		iter := self.edges.Find(selection).Iter()
		result := map[string]interface{}{}
		for iter.Next(&result) {
			send := false
			e := UnpackEdge(result)
			if filter != nil {
				if filter(e) {
					send = true
				}
			} else {
				send = true
			}
			if send {
				o <- e
			}
		}
	}()
	return o
}