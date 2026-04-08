package sqlpersister

import (
	"database/sql"
	"io"
	"os"
)

type client struct {
	db      *sql.DB
	dialect string
	stderr  io.Writer
}

func New(db *sql.DB, dialect string) *client {
	return &client{
		db:      db,
		dialect: dialect,
		stderr:  os.Stderr,
	}
}

func (c *client) Dialect() string {
	return c.dialect
}

func (c *client) Consume(query string) error {
	_, err := c.db.Exec(query)
	if err != nil {
		c.stderr.Write([]byte(query + "\n"))
		return err
	}
	return nil
}

func (c *client) Close() error {
	return c.db.Close()
}
