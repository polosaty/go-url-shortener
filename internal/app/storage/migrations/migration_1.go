package migrations

import (
	"context"
)

func migration1(ctx context.Context, db PgxIface) error {
	_, err := db.Exec(
		ctx,
		`
CREATE TABLE "user" (
    id   bigserial CONSTRAINT user_id_pk PRIMARY KEY,
    uuid UUID
);

CREATE UNIQUE INDEX IF NOT EXISTS user_uuid_uindex on "user"(uuid);

CREATE TABLE url (
    short   VARCHAR(255) CONSTRAINT url_short_pk PRIMARY KEY,
    long    TEXT,
    user_id BIGINT
        CONSTRAINT url_user_id_fk
            references "user"
            ON UPDATE CASCADE ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS url_short_uindex ON url(short);

INSERT INTO revision VALUES(1);  
`)
	return err
}
