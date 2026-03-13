package server

import (
	"github.com/jackc/pgx/v5"
)

func NewQuery(q string, params []any) Query {
	return Query{
		Query:  q,
		Params: params,
		Result: make(chan pgx.Row),
	}
}

func (mgr *Manager) executeQuery(q Query) {
	rows := mgr.dbconn.QueryRow(q.ctx, q.Query, q.Params...)
	q.Result <- rows
}
