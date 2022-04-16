package storage

import (
	"context"
	"fmt"
	pgx "github.com/jackc/pgx/v4"
)

type PG struct {
	db     *pgx.Conn
	memMap *MemoryMap
}

func NewPG(dsn string) (*PG, error) {

	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		//log.Fatal("Unable to connect to database: %v\n", err)
		return nil, fmt.Errorf("Unable to connect to database(dsn=%v): %w", dsn, err)
	}
	//defer conn.Close(context.Background())

	repo := &PG{
		db:     conn,
		memMap: NewMemoryMap(),
	}
	return repo, nil
}

func (d *PG) Close() error {
	return d.db.Close(context.Background())
}

func (d *PG) SaveLongURL(long URL, userID string) (URL, error) {
	short, err := d.memMap.SaveLongURL(long, userID)
	if err != nil {
		return "", err
	}
	return short, nil
}

func (d *PG) GetLongURL(short URL) (URL, error) {
	return d.memMap.GetLongURL(short)
}

func (d *PG) GetUsersUrls(userID string) []URLPair {
	return d.memMap.GetUsersUrls(userID)
}

func (d *PG) Ping() bool {
	//return d.memMap.Ping()
	return d.db.Ping(context.Background()) == nil
}
