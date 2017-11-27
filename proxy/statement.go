package proxy

import (
	"fmt"
	"strings"

	"github.com/lego/mongotunnel/mongo"
	"github.com/lego/mongotunnel/util/context"
	"gopkg.in/mgo.v2/bson"
)

// Handles initial connection negotiation with hardcodeded standards.

func isStatement(ctx *context.Context, query mongo.QueryOp) bool {
	if !strings.HasPrefix(query.Collection, "admin.") && strings.HasSuffix(query.Collection, ".$cmd") {
		return true
	}
	return false
}

func isQuery(ctx *context.Context, query mongo.QueryOp) bool {
	if !isStatement(ctx, query) {
		return false
	}

	// FIXME(joey): Do we need to check the length? I hope not...
	if query.Query[0].Name == "find" {
		return true
	}

	return false
}

func handleQuery(ctx *context.Context, query mongo.QueryOp) (mongo.Op, error) {
	databaseName := strings.Split(query.Collection, ".")[0]
	queryParts := query.Query

	// TODO(joey): These may not be ordered, but this was the order from
	// the Ruby driver. if they are not then using query.Map() would be
	// better.

	// Retrieve query.
	tableName := queryParts[0].Value
	queryParts = queryParts[1:]

	// Retrieve filter.
	var filter map[string]interface{}
	if len(queryParts) > 0 && queryParts[0].Name == "filter" {
		filter = queryParts[0].Value.(bson.D).Map()
		queryParts = queryParts[1:]
	}

	// Retrieve limit.
	limit := -1
	if len(queryParts) > 0 && queryParts[0].Name == "limit" {
		limit = queryParts[0].Value.(int)
		queryParts = queryParts[1:]
	}

	ctx.Log.Debug("query for database=%s table=%s with filter=%v and limit=%d", databaseName, tableName, filter, limit)

	rows, err := ctx.DB.Query(fmt.Sprintf("SELECT * FROM %q.%q", databaseName, tableName))
	if err != nil {
		return nil, err
	}
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	ctx.Log.Debug("columns=%v", cols)

	var replyRows []interface{}

	defer rows.Close()
	for rows.Next() {
		// Create a slice of interface{}'s to represent each column,
		// and a second slice to contain pointers to each item in the columns slice.
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		// Scan the result into the column pointers...
		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		// Create our map, and retrieve the value for each column from the pointers slice,
		// storing it in the map with the name of the column as the key.
		m := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			m[colName] = *val
		}
		replyRows = append(replyRows, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	ctx.Log.Debug("replyRows=%#v", replyRows)
	replyDocuments := bson.D{
		bson.DocElem{
			"cursor",
			bson.D{
				bson.DocElem{"firstBatch", replyRows},
				bson.DocElem{"id", 0},
				bson.DocElem{"ns", fmt.Sprintf("%s.%s", databaseName, tableName)},
			},
		},
		bson.DocElem{"ok", 1},
	}

	return &mongo.ReplyOp{
		Flags:     mongo.ReplyFlagShardConfigStale,
		CursorID:  0,
		FirstDoc:  0,
		ReplyDocs: 1,
		Documents: replyDocuments,
	}, nil
}
