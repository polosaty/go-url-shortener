package migrations

import (
	"context"
)

func migration2(ctx context.Context, db PgxIface) error {
	_, err := db.Exec(
		ctx,
		`
ALTER TABLE url ADD is_deleted bool DEFAULT FALSE;

INSERT INTO revision VALUES(2);  
`)
	return err
}
