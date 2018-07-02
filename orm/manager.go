
package orm

import (
  "reflect"
  "log"
  "github.com/bmeg/arachne/aql"
  "github.com/bmeg/arachne/util/rpc"
  "github.com/bmeg/arachne/protoutil"
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

  for i:= 0; i < t.NumField(); i++ {
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

  wrappedData := protoutil.AsStruct(data)
  vertex := aql.Vertex{Gid:gid, Label:label, Data:wrappedData}
  log.Printf("Adding Vertex: %#v", vertex)

  return nil
}


func (graph *Graph) Get(id string, d interface{}) error {
  t := reflect.TypeOf(d)
  log.Printf("Finding %s", t.Name)
  return nil
}
