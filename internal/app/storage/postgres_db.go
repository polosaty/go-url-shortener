package storage

import (
	"context"
	"errors"
	"fmt"
	pgx "github.com/jackc/pgx/v4"
	"strconv"
)

type PG struct {
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
	_, err := d.db.Exec(
		context.Background(),
		`CREATE table IF NOT EXISTS "revision" (version BIGSERIAL CONSTRAINT revision_version_pk PRIMARY KEY)`)
	if err != nil {
		return fmt.Errorf("cannot get or create table revision: %w", err)
	}
	var version int64
	err = d.db.QueryRow(
		context.Background(),
		"SELECT version FROM revision ORDER BY version DESC LIMIT 1").
		Scan(&version)

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
		`create table "user" (
    id   bigserial
        constraint user_id_pk
            primary key,
    uuid uuid
);

alter table "user"
owner to postgres;

create unique index user_uuid_uindex
    on "user"(uuid);

create table url (
    short   varchar(255)
        constraint url_short_pk
            primary key,
    long    text,
    user_id bigint
        constraint url_user_id_fk
            references "user"
            on update cascade on delete set null
);

create unique index url_short_uindex
    on url(short);

INSERT INTO revision values(1);  
`)
	return err
}

func (d *PG) Close() error {
	return d.db.Close(context.Background())
}

func (d *PG) SaveLongURL(long URL, userID string) (URL, error) {

	short, err := Hash(long.S())
	if err != nil {
		return "", fmt.Errorf("cantnot generate short url: %w", err)
	}

	shortURL := URL(strconv.FormatUint(uint64(short), 16))

	var userPK int64

	userPK, err = d.getOrCreateUser(userID)
	if err != nil {
		return "", fmt.Errorf("cantnot get or create user: %w", err)
	}

	_, err = d.db.Exec(context.Background(),
		`INSERT INTO "url" ("short", "long", "user_id") VALUES($1, $2, $3) 
			ON CONFLICT ("short") DO NOTHING`, shortURL, long, userPK)

	if err != nil {
		return "", fmt.Errorf("cantnot save url to db: %w", err)
	}

	return shortURL, nil
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

func (d *PG) GetUsersUrls(userID string) []URLPair {
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
