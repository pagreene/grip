
package gdbi

import (
  "log"
  "testing"
  "github.com/bmeg/arachne/aql"
  "github.com/golang/protobuf/jsonpb"
)

var edgeGenStr1 = `
{
    "label" : "related", "unzip" : {
        "field" : "$.data", "match" : [{"field" : "gid", "value" : ["$[0]"]}], "data" : "$[1]"
    }
}
`

var vertexStr1 = `
{
  "gid" : "test",
  "label" : "Person",
  "data" : {
    "hello" : { "v" : 1},
    "world" : { "v" : 1}
  },
  "gen" : [
    {
        "label" : "related", "unzip" : {
            "field" : "$.data", "match" : [{"field" : "gid", "value" : ["$[0]"]}], "data" : "$[1]"
        }
    }
  ]
}
`

func TestParsing(t *testing.T) {
  g := aql.EdgeGen{}
  jsonpb.UnmarshalString(edgeGenStr1, &g)

  v := aql.Vertex{}
  jsonpb.UnmarshalString(vertexStr1, &v)
}


func TestEdgeGen(t *testing.T) {
  v := aql.Vertex{}
  jsonpb.UnmarshalString(vertexStr1, &v)
  log.Printf("%s", v)

  edges := EdgeGenerate(v, []string{"related"})

  for _, e := range edges {
    log.Printf("%s : %s : %s", e.Gid, e.Label, e.Data)
  }

}
