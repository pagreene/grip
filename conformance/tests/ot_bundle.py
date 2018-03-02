


def test_bundle_filter(O):
    errors = []

    edges = {}
    for i in range(100):
        O.addVertex("dstVertex%d" % i, "person")
        edges["dstVertex%d" % i] = {"val" : i}

    O.addVertex("srcVertex", "person", edges, gen=[
        {
            "label" : "related", "unzip" : {
                "field" : "$.", "match" : [{"field" : "gid", "value" : ["$[0]"]}], "data" : "$[1]"
            }
        }
    ])

    count = 0
    for i in O.query().V("srcVertex").outgoingEdge("related").filter("function(x) { return x.val > 50; }").outgoing().execute():
        count += 1

    if count != 49:
        errors.append("Fail: Bundle Filter %s != %s" % (count, 49))

    count = 0
    for i in O.query().V("srcVertex").outgoing("related").execute():
        count += 1
    if count != 100:
        errors.append("Fail: Bundle outgoing %s != %s" % (count, 100))


    return errors
