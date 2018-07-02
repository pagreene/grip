package orm

import (
	"fmt"
	"github.com/bmeg/arachne/aql"
	"github.com/bmeg/arachne/protoutil"
	"github.com/bmeg/arachne/util/rpc"
	"log"
	"reflect"
)

type Graph struct {
	client aql.Client
	graph  string
}

func NewGraph(host, graph string) *Graph {
	conn, _ := aql.Connect(rpc.ConfigWithDefaults(host), true)
	return &Graph{conn, graph}
}

func (graph *Graph) Add(d interface{}) error {
	t := reflect.TypeOf(d)
	v := reflect.ValueOf(d)
	var gid string
	label := t.Name()
	data := map[string]interface{}{}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		dataField := true
		//if the field defines the gid
		if tag, ok := f.Tag.Lookup("graph"); ok {
			if tag == "ID" {
				f := v.Field(i)
				if s, ok := f.Interface().(string); ok {
					gid = s
				}
				dataField = false
			}
		}
		if dataField {
			fv := v.Field(i)
			data[f.Name] = fv.Interface()
		}
	}
	if gid == "" {
		return fmt.Errorf("gid not found")
	}

	wrappedData := protoutil.AsStruct(data)
	vertex := aql.Vertex{Gid: gid, Label: label, Data: wrappedData}
	log.Printf("Adding Vertex: %#v", vertex)
	graph.client.AddVertex(graph.graph, &vertex)
	return nil
}

func (graph *Graph) Get(id string, d interface{}) error {
	vptr := reflect.ValueOf(d)
	if vptr.Kind() != reflect.Ptr {
		return fmt.Errorf("Need pointer")
	}
	v := vptr.Elem()
	t := reflect.TypeOf(v.Interface())

	vertex, err := graph.client.GetVertex(graph.graph, id)
	if err != nil {
		return err
	}
	log.Printf("%s", vertex)
	data := protoutil.AsMap(vertex.Data)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		dataField := true
		//if the field defines the gid
		if tag, ok := f.Tag.Lookup("graph"); ok {
			if tag == "ID" {
				f := v.Field(i)
				f.SetString(id)
				dataField = false
			}
		}
		if dataField {
			//TODO: these needs to be a lot of type checking logic here
			fv := v.Field(i)
			fv.Set(reflect.ValueOf(data[f.Name]))
		}
	}
	return nil
}
