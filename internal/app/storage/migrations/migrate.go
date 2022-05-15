package migrations

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"log"
)

type PgxIface interface {
	Begin(context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
	Ping(context.Context) error
	//Prepare(context.Context, string, string) (*pgconn.StatementDescription, error)
	//Close(context.Context) error
	Close()
}

type migration func(ctx context.Context, db PgxIface) error

func Migrate(ctx context.Context, db PgxIface) error {
	_, err := db.Exec(
		ctx, `CREATE TABLE IF NOT EXISTS "revision" (version BIGSERIAL CONSTRAINT revision_version_pk PRIMARY KEY)`)
	if err != nil {
		return fmt.Errorf("cannot get or create table revision: %w", err)
	}
	var version int
	err = db.QueryRow(
		ctx, "SELECT version FROM revision ORDER BY version DESC LIMIT 1").Scan(&version)

	if err != nil &&
		!(errors.Is(err, pgx.ErrNoRows)) {
		return fmt.Errorf("cannot get version: %w", err)
	}

	migrations := []migration{
		migration1,
		migration2,
	}

	for v, m := range migrations {
		if version < (v + 1) {
			log.Println("migrate database to version: ", v+1)
			if err = m(ctx, db); err != nil {
				return err
			}
		}
	}

	return nil
}
