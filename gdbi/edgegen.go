
package gdbi

import (
  "log"
  "github.com/bmeg/arachne/aql"
)

func EdgeGenerate(v aql.Vertex, labels []string) []aql.Edge {
  out := []aql.Edge{}

  for _, g := range v.Gen {
    if len(labels) == 0 || contains(labels, g.Label) {
      if v, ok := g.Build.(*aql.EdgeGen_Unzip); ok {
    		log.Printf("Unzip: %s", v)
      } else if v, ok := g.Build.(*aql.EdgeGen_Match); ok {
        log.Printf("Match: %s", v)
      } else if v, ok := g.Build.(*aql.EdgeGen_Range); ok {
        log.Printf("Range: %s", v)
      }
    }
  }

  return out;
}
