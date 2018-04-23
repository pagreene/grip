package gdbi

import (
	"github.com/bmeg/arachne/aql"
	"github.com/golang/protobuf/jsonpb"
	"log"
	"testing"
)

var edgeGenStr1 = `
{
    "label" : "related",
		"from_label" : "Person",
		"to_label" : "Person"
		"unzip" : {
        "field" : "$.data", "match" : [{"field" : "gid", "value" : "$.[0]"}], "data" : "$[1]"
    }
}
`

var edgeGenStr2 = `
"gen" : [
	{
			"label" : "related", "unzip" : {
					"field" : "$.data", "match" : [{"field" : "gid", "value" : ["$.[0]"], "op":"EQ"}], "data" : "$[1]"
			}
	}
]
`

var vertexStr1 = `
{
  "gid" : "test",
  "label" : "Person",
  "data" : {
    "hello" : { "v" : 1},
    "world" : { "v" : 1}
  }
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

	g := aql.EdgeGen{}
	jsonpb.UnmarshalString(edgeGenStr1, &g)

	queries := EdgeGenerate(&v, g)

	for _, q := range queries {
		log.Printf("%s", q)
	}

}
