package test

import (
	"encoding/json"
	"fmt"
	"log"
	//"math"
	"testing"

	"github.com/bmeg/arachne/kvindex"
)

var numDocs = `[
{"value" : 1},
{"value" : 2},
{"value" : 3},
{"value" : -42},
{"value" : 3.14},
{"value" : 70.1},
{"value" : 400.7},
{"value" : 3.14},
{"value" : 0.2},
{"value" : -0.2},
{"value" : -42},
{"value" : -150},
{"value" : -200},
{"value" : -15},
{"value" : -42},
{"value" : 3.14}
]`

func TestFloatSorting(t *testing.T) {
	resetKVInterface()
	idx := kvindex.NewIndex(kvdriver)

	newFields := []string{"value"}
	for _, s := range newFields {
		idx.AddField(s)
	}

	data := []map[string]interface{}{}
	json.Unmarshal([]byte(numDocs), &data)
	for i, d := range data {
		log.Printf("Adding: %s", d)
		idx.AddDoc(fmt.Sprintf("%d", i), d)
	}

	//idx.AddDoc("a", map[string]interface{}{"value": math.Inf(1)})
	//idx.AddDoc("b", map[string]interface{}{"value": math.Inf(-1)})

	last := -10000.0
	log.Printf("Scanning")
	for d := range idx.FieldNumbers("value") {
		if d < last {
			t.Errorf("Incorrect field return order: %f < %f", d, last)
		}
		last = d
		log.Printf("Scan, %f", d)
	}

	log.Printf("Min %f", idx.FieldTermNumberMin("value"))
	log.Printf("Max %f", idx.FieldTermNumberMax("value"))

	if v := idx.FieldTermNumberMin("value"); v != -200.0 {
		t.Errorf("Incorrect Min %f != %f", v, -200.0)
	}
	if v := idx.FieldTermNumberMax("value"); v != 400.7 {
		t.Errorf("Incorrect Max: %f != %f", v, 400.7)
	}

}

func TestFloatRange(t *testing.T) {
	resetKVInterface()
	idx := kvindex.NewIndex(kvdriver)

	newFields := []string{"value"}
	for _, s := range newFields {
		idx.AddField(s)
	}

	data := []map[string]interface{}{}
	json.Unmarshal([]byte(numDocs), &data)
	for i, d := range data {
		//log.Printf("Adding: %s", d)
		idx.AddDoc(fmt.Sprintf("%d", i), d)
	}

	for d := range idx.FieldTermNumberRange("value", 5, 100) {
		if d.Number < 5 || d.Number > 100 {
			t.Errorf("Out of Range Value: %f", d.Number)
		}
	}

	for d := range idx.FieldTermNumberRange("value", -100, 10) {
		if d.Number < -100 || d.Number > 10 {
			t.Errorf("Out of Range Value: %f", d.Number)
		}
		if d.Number == -42 && d.Count != 3 {
			t.Errorf("Incorrect term count")
		}
		if d.Number == 3.14 && d.Count != 3 {
			t.Errorf("Incorrect term count")
		}
		//log.Printf("%#v", d)
	}

}
