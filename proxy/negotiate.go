package proxy

import (
	"github.com/lego/mproxy/mongo"
	"github.com/lego/mproxy/util/context"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2/bson"
)

// Handles initial connection negotiation messages and replies with hardcodeded values.

func isNegotiation(ctx *context.Context, query mongo.QueryOp) bool {
	if query.Collection == "admin.$cmd" {
		return true
	}
	return false
}

func createNegotiationReply(ctx *context.Context, query mongo.QueryOp) (mongo.Op, error) {
	// ctx.Log.Warn("negotiation query docLen=%d", len(query.Query))
	// for i, doc := range query.Query {
	// 	ctx.Log.Warn("query[%d]=%v", i, doc)
	// }
	// if len(query.Query) == 0 {
	doc := query.Query.Map()
	if doc["ismaster"] == 1 {
		return &mongo.ReplyOp{
			Flags:     mongo.ReplyFlagShardConfigStale,
			CursorID:  0,
			FirstDoc:  0,
			ReplyDocs: 1,
			Documents: bson.D{
				bson.DocElem{"ismaster", true},
				bson.DocElem{"maxBsonObjectSize", 16777216},
				bson.DocElem{"maxMessageSizeBytes", 48000000},
				bson.DocElem{"maxWriteBatchSize", 1000},
				bson.DocElem{"localTime", "2017-11-10 14:39:34.347 -0500 EST"},
				bson.DocElem{"maxWireVersion", 5},
				bson.DocElem{"minWireVersion", 0},
				// Whoa, we can declare the server as readonly? By
				// observation, the ruby driver does not respect it
				// though.
				bson.DocElem{"readOnly", false},
				bson.DocElem{"ok", 1},
			},
		}, nil
	}
	// }
	return nil, errors.Errorf("could not identify negotiation query")
}
