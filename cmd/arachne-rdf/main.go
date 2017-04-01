
package main


import (
  "os"
  "io"
  "log"
  "flag"
  //"fmt"
  "github.com/knakk/rdf"
  ophion "github.com/bmeg/ophion/client/go"
  "compress/gzip"
)


func main() {
	flag.Parse()

  f, err := os.Open(flag.Arg(0))
  if err != nil {
    log.Printf("Error: %s", err)
    os.Exit(1)
  }
  server := flag.Arg(1)
  conn, err := ophion.Connect(server)
  if err != nil {
    log.Printf("%s", err)
    os.Exit(1)
  }

  vert_map := map[string]int{}

  count := 0
  fz, _ := gzip.NewReader(f)
  dec := rdf.NewTripleDecoder(fz, rdf.RDFXML)
  for triple, err := dec.Decode(); err != io.EOF; triple, err = dec.Decode() {
    subj := triple.Subj.String()
    if _, ok := vert_map[subj]; !ok {
      ophion.Query(conn).AddV(subj).Run()
      vert_map[subj] = 1
    }
    if triple.Obj.Type() == rdf.TermLiteral {
      ophion.Query(conn).V(subj).Property(triple.Pred.String(), triple.Obj.String()).Run()
    } else {
      obj := triple.Obj.String()
      if _, ok := vert_map[obj]; !ok {
        ophion.Query(conn).AddV(obj).Run()
        vert_map[obj] = 1
      }
      ophion.Query(conn).V(subj).AddE(triple.Pred.String()).To(obj).Run()
    }
    if count % 1000 == 0 {
      log.Printf("Processed %d triples", count)
    }
    count++
  }
}
