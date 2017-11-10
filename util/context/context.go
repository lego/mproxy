package context

import (
	"database/sql"

	"github.com/lego/mproxy/util/log"
)

type Context struct {
	Log log.Logger
	DB  *sql.DB
}

func NewContext(log log.Logger) *Context {
	return &Context{
		Log: log,
	}
}

func (ctx *Context) SetLogger(log log.Logger) {
	ctx.Log = log
}

func (ctx *Context) SetDB(db *sql.DB) {
	ctx.DB = db
}
