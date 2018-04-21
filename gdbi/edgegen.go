package gdbi

import (
	"github.com/bmeg/arachne/aql"
	"github.com/bmeg/arachne/jsonpath"
	"github.com/bmeg/arachne/protoutil"
	"log"
	"strings"
)

func VertexToMap(a aql.Vertex) map[string]interface{} {
	d := protoutil.AsMap(a.Data)
	out := map[string]interface{}{
		"gid":   a.Gid,
		"label": a.Label,
		"data":  d,
	}
	return out
}

type FieldMatch struct {
	Field string
	Value interface{}
	Op    aql.Condition
}

func contains(a []string, v string) bool {
	for _, i := range a {
		if i == v {
			return true
		}
	}
	return false
}

func BuildMatch(match []*aql.VertexMatch, data interface{}) []FieldMatch {
	out := []FieldMatch{}
	for _, m := range match {
		s := make([]string, len(m.Value))
		for i := range m.Value {
			t, err := jsonpath.Get(data, m.Value[i])
			if err != nil {
				log.Printf("Path Error: %s %s %s", m.Value[i], data, err)
			}
			s[i] = t.(string)
		}
		v := strings.Join(s, "")
		out = append(out, FieldMatch{Field: m.Field, Value: v, Op: m.Op})
	}
	return out
}

func EdgeGenerate(v aql.Vertex, labels []string) []aql.Edge {
	out := []aql.Edge{}

	for _, g := range v.Gen {
		if len(labels) == 0 || contains(labels, g.Label) {
			if u, ok := g.Build.(*aql.EdgeGen_Unzip); ok {
				m := VertexToMap(v)
				field, _ := jsonpath.Get(m, u.Unzip.Field)
				if fieldMap, ok := field.(map[string]interface{}); ok {
					for k, v := range fieldMap {
						m := BuildMatch(u.Unzip.Match, []interface{}{k, v})
						log.Printf("MatchCriteria: %s", m)
						//These match critieria would then be passed into an index engine
						//to find matching dest verts
					}
				}
			} else if v, ok := g.Build.(*aql.EdgeGen_Match); ok {
				log.Printf("Match: %s", v)
			} else if v, ok := g.Build.(*aql.EdgeGen_Range); ok {
				log.Printf("Range: %s", v)
			}
		}
	}

	return out
}
