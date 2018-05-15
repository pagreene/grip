from __future__ import absolute_import

import aql

def test_index(O):
    errors = []

    O.addIndex("Person", "name")
    O.addIndex("Person", "age")

    O.addVertex("1", "Person", {"name": "marko", "age": 29})
    O.addVertex("2", "Person", {"name": "vadas", "age": 27})
    O.addVertex("3", "Software", {"name": "lop", "lang": "java"})
    O.addVertex("4", "Person", {"name": "josh", "age": 32})
    O.addVertex("5", "Software", {"name": "ripple", "lang": "java"})
    O.addVertex("6", "Person", {"name": "peter", "age": 35})
    O.addVertex("7", "Person", {"name": "marko", "age": 35})

    O.addEdge("1", "3", "created", {"weight": 0.4})
    O.addEdge("1", "2", "knows", {"weight": 0.5})
    O.addEdge("1", "4", "knows", {"weight": 1.0})
    O.addEdge("4", "3", "created", {"weight": 0.4})
    O.addEdge("6", "3", "created", {"weight": 0.2})
    O.addEdge("4", "5", "created", {"weight": 1.0})

    found = False
    for i in O.query().V().where(aql.eq("_label", "Person")).where(aql.eq("name", "marko")):
        print i


    return errors
