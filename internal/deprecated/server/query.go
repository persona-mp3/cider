package server

import (
	"github.com/jackc/pgx/v5"
)

type Query struct {
	query  string
	params []any
	result chan pgx.Row
}

func NewQuery(q string, params []any) Query {
	return Query{
		query:  q,
		params: params,
		result: make(chan pgx.Row),
	}
}
