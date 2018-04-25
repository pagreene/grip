package gdbi

import (
	"github.com/bmeg/arachne/aql"
	"github.com/bmeg/arachne/jsonpath"
	"github.com/bmeg/arachne/protoutil"
	"log"
	//"strings"
)

func VertexToMap(a *aql.Vertex) map[string]interface{} {
	d := protoutil.AsMap(a.Data)
	out := map[string]interface{}{
		"gid":   a.Gid,
		"label": a.Label,
		"data":  d,
	}
	return out
}

type FieldMatch struct {
	Gid   string
	Field string
	Value interface{}
	Op    aql.Condition
}

func BuildMatch(match []*aql.VertexMatch, data interface{}) []FieldMatch {
	out := []FieldMatch{}
	for _, m := range match {
		log.Printf("Field value %s", m.Value)
		t, err := jsonpath.Get(data, m.Value)
		if err != nil {
			log.Printf("Path Error: %s %s %s", m.Value, data, err)
		}
		out = append(out, FieldMatch{Field: m.Field, Value: t, Op: m.Op})
	}
	return out
}

func EdgeGenerate(vert *aql.Vertex, gen aql.EdgeGen) []FieldMatch {
	out := []FieldMatch{}
	if u, ok := gen.Build.(*aql.EdgeGen_Unzip); ok {
		m := VertexToMap(vert)
		field, _ := jsonpath.Get(m, u.Unzip.Field)
		if fieldMap, ok := field.(map[string]interface{}); ok {
			for k, v := range fieldMap {
				out = BuildMatch(u.Unzip.Match, []interface{}{k, v})
			}
		}
	} else if v, ok := gen.Build.(*aql.EdgeGen_Match); ok {
		log.Printf("Match: %s", v)
	} else if v, ok := gen.Build.(*aql.EdgeGen_Range); ok {
		log.Printf("Range: %s", v)
	}
	return out
}
