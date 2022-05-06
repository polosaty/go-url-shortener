package storage

import (
	"context"
	"errors"
	"fmt"
	pgx "github.com/jackc/pgx/v4"
	"go-url-shortener/internal/app/storage/migrations"
	"log"
	"sync"
	"time"
)

type delayedUserUrlsDeleter struct {
	mu       sync.RWMutex
	userChan map[int64]chan URL
	done     chan struct{}
}

type PG struct {
	Repository
	db             *pgx.Conn
	delayedDeleter *delayedUserUrlsDeleter
}

var ErrNoRows = pgx.ErrNoRows

func newDeleteUserUrls() *delayedUserUrlsDeleter {
	return &delayedUserUrlsDeleter{
		userChan: make(map[int64]chan URL),
	}
}

func NewPG(dsn string) (*PG, error) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		//log.Fatal("Unable to connect to database: %v\n", err)
		return nil, fmt.Errorf("unable to connect to database(dsn=%v): %w", dsn, err)
	}

	repo := &PG{
		db:             conn,
		delayedDeleter: newDeleteUserUrls(),
	}
	err = migrations.Migrate(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("cannot apply migrations: %w", err)
	}
	return repo, nil
}

func (d *PG) Close() error {
	d.delayedDeleter.Stop()
	return d.db.Close(context.Background())
}

func (d *delayedUserUrlsDeleter) Stop() {
	d.done <- struct{}{}
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
	log.Println("by userUUID:", userUUID, "got userPK:", userPK)
	return

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
	tx, err := d.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot begin transaction: %w", err)
	}
	_, err = tx.Exec(ctx, `CREATE TEMP TABLE tmp_table ON COMMIT DROP AS SELECT * FROM "url" WITH NO DATA`)
	if err != nil {
		return nil, fmt.Errorf("cannot create temp table: %w", err)
	}
	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"tmp_table"},
		[]string{"short", "long", "user_id", "is_deleted"},
		pgx.CopyFromSlice(len(copyFromRows), func(i int) ([]interface{}, error) {
			row := copyFromRows[i]
			return []interface{}{row.Short, row.Long, row.UserPK, false}, nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot insert rows to temp table: %w", err)
	}

	_, err = tx.Exec(ctx, `INSERT INTO "url" SELECT DISTINCT ON (short) * FROM tmp_table
ON CONFLICT ("short")
DO UPDATE SET long = EXCLUDED.long, is_deleted = false
WHERE url.user_id = EXCLUDED.user_id and url.short = EXCLUDED.short`)
	if err != nil {
		return nil, fmt.Errorf("cannot insert rows from temp table: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return result, nil
}

func (d *PG) GetLongURL(short URL) (URL, error) {
	var long URL
	var isDeleted bool
	err := d.db.QueryRow(context.Background(),
		`SELECT long, is_deleted FROM url WHERE short = $1 LIMIT 1`, short).
		Scan(&long, &isDeleted)
	if errors.Is(err, ErrNoRows) {
		return "", err
	}
	if isDeleted {
		return "", ErrDeletedURL
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

func (d *PG) DeleteUsersURLs(userUUID string, shortUrls ...URL) (err error) {
	userPK, err := d.getOrCreateUser(userUUID)
	if err != nil {
		return fmt.Errorf("cannot get or create user: %w", err)
	}
	_, err = d.db.Exec(context.Background(),
		`UPDATE "url" SET is_deleted = true WHERE short = any($1) and user_id = $2`, shortUrls, userPK)
	//tag.RowsAffected()
	return err
}

func (d *delayedUserUrlsDeleter) makeChan(db *PG, userPK int64, userUUID string) chan URL {
	channel := make(chan URL)

	d.mu.Lock()
	defer d.mu.Unlock()
	d.userChan[userPK] = channel

	go func() {
		urls := make([]URL, 0, 1000)
		ticker := time.NewTicker(time.Second * 2)
		for {
			// собираем url из канала и либо, по таймауту либо, по достижении 1000 шт отправляем в базу
			select {
			case url := <-channel:
				urls = append(urls, url)
				if len(urls) >= 1000 {
					err := db.DeleteUsersURLs(userUUID, urls...)
					if err != nil {
						log.Printf("error in delayed delete: %v", err)
						continue
					}
					urls = urls[:0]
				}
			case <-ticker.C:
				if len(urls) < 1 {
					//TODO: можно удалять канал если он долго пустой
					continue
				}
				err := db.DeleteUsersURLs(userUUID, urls...)
				if err != nil {
					log.Printf("error in delayed delete: %v", err)
					continue
				}
				urls = urls[:0]

			case <-d.done:
				close(channel)
				return
			}

		}
	}()

	return channel
}

func (d *delayedUserUrlsDeleter) PostUrlsForDelete(db *PG, userPK int64, userUUID string, shortUrls ...URL) {
	//проверить есть ли канал для userId если нет - создать
	d.mu.RLock()
	defer d.mu.RUnlock()
	channel, found := d.userChan[userPK]

	if !found {
		d.mu.RUnlock()
		channel = d.makeChan(db, userPK, userUUID)
		d.mu.RLock()
	}
	go func() {
		for _, url := range shortUrls {
			channel <- url
		}
	}()
}

func (d *PG) DelayedDeleteUsersURLs(userID string, shortUrls ...URL) (err error) {
	userPK, err := d.getOrCreateUser(userID)
	if err != nil {
		return fmt.Errorf("cannot get or create user: %w", err)
	}
	d.delayedDeleter.PostUrlsForDelete(d, userPK, userID, shortUrls...)
	return nil
}
