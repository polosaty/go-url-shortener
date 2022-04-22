package storage

import (
	"context"
	"errors"
	"fmt"
	pgx "github.com/jackc/pgx/v4"
	"log"
)

type PG struct {
	Repository
	db     *pgx.Conn
	memMap *MemoryMap
}

var ErrNoRows = pgx.ErrNoRows

func NewPG(dsn string) (*PG, error) {

	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		//log.Fatal("Unable to connect to database: %v\n", err)
		return nil, fmt.Errorf("unable to connect to database(dsn=%v): %w", dsn, err)
	}
	//defer conn.Close(context.Background())

	repo := &PG{
		db:     conn,
		memMap: NewMemoryMap(),
	}
	err = repo.Migrate()
	if err != nil {
		return nil, fmt.Errorf("cannot apply migrations: %w", err)
	}
	return repo, nil
}

func (d *PG) Migrate() error {
	ctx := context.Background()

	_, err := d.db.Exec(
		ctx, `CREATE TABLE IF NOT EXISTS "revision" (version BIGSERIAL CONSTRAINT revision_version_pk PRIMARY KEY)`)
	if err != nil {
		return fmt.Errorf("cannot get or create table revision: %w", err)
	}
	var version int64
	err = d.db.QueryRow(
		ctx, "SELECT version FROM revision ORDER BY version DESC LIMIT 1").Scan(&version)

	if err != nil &&
		!(errors.Is(err, ErrNoRows)) {
		return fmt.Errorf("cannot get version: %w", err)
	}
	if version < 1 {
		return d.migration1()
	}
	return nil
}

func (d *PG) migration1() error {
	_, err := d.db.Exec(
		context.Background(),
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

func (d *PG) Close() error {
	return d.db.Close(context.Background())
}

func (d *PG) SaveLongURL(long URL, userID string) (URL, error) {
	shortURL, err := makeShort(long)
	if err != nil {
		return "", fmt.Errorf("cannot generate short url: %w", err)
	}

	var userPK int64
	userPK, err = d.getOrCreateUser(userID)
	if err != nil {
		return "", fmt.Errorf("cannot get or create user: %w", err)
	}

	ct, err := d.db.Exec(context.Background(),
		`INSERT INTO "url" ("short", "long", "user_id") VALUES($1, $2, $3) 
			ON CONFLICT ("short") DO NOTHING RETURNING "short"`, shortURL, long, userPK)

	if err != nil {
		return "", fmt.Errorf("cannot save url to db: %w", err)
	}
	if ct.RowsAffected() < 1 {
		return shortURL, NewConflictURLError(shortURL, ErrConflictURL)
	}

	return shortURL, nil
}

func (d *PG) SaveLongBatchURL(longURLS []CorrelationLongPair, userID string) ([]CorrelationShortPair, error) {
	ctx := context.Background() // TODO: take context from request
	userPK, err := d.getOrCreateUser(userID)
	if err != nil {
		return nil, fmt.Errorf("cannot get or create user: %w", err)
	}

	//insertUrlsStmt, err := d.db.Prepare(
	//	ctx,
	//	"insert_urls_stmt",
	//	`INSERT INTO "url" ("short", "long", "user_id") VALUES($1, $2, $3)
	//		ON CONFLICT ("short") DO NOTHING`)
	//, shortURL, long, userPK
	//if err != nil {
	//	return nil, fmt.Errorf("cannot prepare insert: %w", err)
	//}

	result := make([]CorrelationShortPair, 0, len(longURLS))

	type rowStruct struct {
		Short  string
		Long   string
		UserPK int64
	}
	copyFromRows := make([]rowStruct, 0, len(longURLS))
	for _, p := range longURLS {

		shortURL, err := makeShort(p.LongURL)
		if err != nil {
			log.Printf("cannot generate short url: %v for url: %v", err, p.LongURL)
			continue
		}

		copyFromRows = append(copyFromRows, rowStruct{shortURL.S(), p.LongURL.S(), userPK})
		result = append(result, CorrelationShortPair{p.CorrelationID, shortURL})
	}
	//FIXME: cannot insert rows: ERROR: duplicate key value violates unique constraint "url_short_pk" (SQLSTATE 23505)
	_, err = d.db.CopyFrom(
		ctx,
		pgx.Identifier{"url"},
		[]string{"short", "long", "user_id"},
		pgx.CopyFromSlice(len(copyFromRows), func(i int) ([]interface{}, error) {
			row := copyFromRows[i]
			return []interface{}{row.Short, row.Long, row.UserPK}, nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot insert rows: %w", err)
	}

	return result, nil
}

func (d *PG) getOrCreateUser(userUUID string) (userPK int64, err error) {
	err = d.db.QueryRow(context.Background(),
		`SELECT id FROM "user" WHERE "uuid"=$1 LIMIT 1`, userUUID).
		Scan(&userPK)
	if errors.Is(err, ErrNoRows) {
		err = d.db.QueryRow(context.Background(),
			`INSERT INTO "user" (uuid) VALUES($1) 
			ON CONFLICT (uuid) DO NOTHING RETURNING id`, userUUID).
			Scan(&userPK)
		if err != nil {
			return
		}
	}

	return

}

func (d *PG) GetLongURL(short URL) (URL, error) {
	var long URL
	err := d.db.QueryRow(context.Background(),
		`SELECT long FROM url WHERE short = $1 LIMIT 1`, short).
		Scan(&long)
	if errors.Is(err, ErrNoRows) {
		return "", err
	}
	return long, nil
}

func (d *PG) GetUsersURLs(userID string) []URLPair {
	rows, err := d.db.Query(context.Background(),
		`SELECT "long", "short" FROM "url" 
		JOIN "user" ON "user".id = "url".user_id
		WHERE "user".uuid = $1`, userID)

	if err != nil {
		return nil
	}
	var urlPairs []URLPair

	for rows.Next() {
		var v URLPair
		err = rows.Scan(&v.LongURL, &v.ShortURL)
		if err != nil {
			return nil
		}
		urlPairs = append(urlPairs, v)
	}
	return urlPairs
}

func (d *PG) Ping() bool {
	return d.db.Ping(context.Background()) == nil
}
