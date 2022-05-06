package migrations

import (
	"context"
	"github.com/jackc/pgx/v4"
)

func migration2(ctx context.Context, db *pgx.Conn) error {
	_, err := db.Exec(
		ctx,
		`
ALTER TABLE url ADD is_deleted bool DEFAULT FALSE;

INSERT INTO revision VALUES(2);  
`)
	return err
}
