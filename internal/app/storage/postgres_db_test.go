package storage

import (
	"fmt"
	"github.com/pashagolub/pgxmock"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPG_SaveLongURL(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	userUUID := "882de4ff-11d0-48ea-9674-7ac516c89baa"
	longURL := URL("long_url")
	expectedShortURL := URL("6db64c5d")
	mock.ExpectQuery(`SELECT id FROM \"user\" WHERE \"uuid\"\=\$1 LIMIT 1`).
		WithArgs(userUUID).
		WillReturnRows(
			mock.NewRows([]string{"id"}).
				AddRow(int64(123)))
	mock.ExpectExec(`INSERT INTO "url" (.*) VALUES(.*) ON CONFLICT \("short"\) DO NOTHING RETURNING "short"`).
		WithArgs(expectedShortURL, longURL, int64(123)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	//defer mock.Close(context.Background())
	defer mock.Close()
	type fields struct {
		db             PgxIface
		delayedDeleter *delayedUserUrlsDeleter
	}
	type args struct {
		long   URL
		userID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    URL
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Test #1 Success post long and get short",
			fields: fields{
				db:             mock,
				delayedDeleter: nil,
			},
			args: args{long: longURL, userID: userUUID},
			want: expectedShortURL,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &PG{
				//Repository:     tt.fields.Repository,
				db:             tt.fields.db,
				delayedDeleter: tt.fields.delayedDeleter,
			}

			got, err := d.SaveLongURL(tt.args.long, tt.args.userID)
			if tt.wantErr != nil && !tt.wantErr(t, err, fmt.Sprintf("SaveLongURL(%v, %v)", tt.args.long, tt.args.userID)) {
				return
			}
			assert.Equalf(t, tt.want, got, "SaveLongURL(%v, %v)", tt.args.long, tt.args.userID)
		})
	}
}

func TestPG_GetUsersURLs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	userUUID := "882de4ff-11d0-48ea-9674-7ac516c89baa"
	longURL := URL("long_url")
	expectedShortURL := URL("6db64c5d")
	mock.ExpectQuery(`SELECT "long", "short" FROM "url" JOIN "user"`).
		WithArgs(userUUID).
		WillReturnRows(
			mock.NewRows([]string{"long", "short"}).
				AddRow(longURL, expectedShortURL))

	type fields struct {
		db             PgxIface
		delayedDeleter *delayedUserUrlsDeleter
	}
	type args struct {
		userID string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []URLPair
	}{
		{
			name: "Test #1 Success get users posts",
			fields: fields{
				db: mock,
			},
			args: args{userUUID},
			want: []URLPair{{expectedShortURL, longURL}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &PG{
				db:             tt.fields.db,
				delayedDeleter: tt.fields.delayedDeleter,
			}
			assert.Equalf(t, tt.want, d.GetUsersURLs(tt.args.userID), "GetUsersURLs(%v)", tt.args.userID)
		})
	}
}

func TestPG_DelayedDeleteUsersURLs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	shortUrlsForDelete := []URL{"6db64c5d", "6db64c5e", "6db64c5f"}
	userUUID := "882de4ff-11d0-48ea-9674-7ac516c89baa"
	userPK := int64(123)

	mock.ExpectQuery(`SELECT id FROM \"user\" WHERE \"uuid\"\=\$1 LIMIT 1`).
		WithArgs(userUUID).
		WillReturnRows(
			mock.NewRows([]string{"id"}).
				AddRow(userPK))

	mock.ExpectQuery(`SELECT id FROM \"user\" WHERE \"uuid\"\=\$1 LIMIT 1`).
		WithArgs(userUUID).
		WillReturnRows(
			mock.NewRows([]string{"id"}).
				AddRow(userPK))

	mock.ExpectExec(`UPDATE "url" SET is_deleted = true WHERE short = any\(\$1\) and user_id = \$2`).
		WithArgs(shortUrlsForDelete, userPK).
		WillReturnResult(pgxmock.NewResult("UPDATE", 3))
	//mock.ExpectCommit()

	type fields struct {
		db PgxIface
	}
	type args struct {
		userID    string
		shortUrls []URL
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "Test #1 Success delete urls",
			fields: fields{
				db: mock,
			},
			args: args{userUUID, shortUrlsForDelete},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &PG{
				db:             tt.fields.db,
				delayedDeleter: newDeleteUserUrls(),
			}
			err := d.DelayedDeleteUsersURLs(tt.args.userID, tt.args.shortUrls...)
			assert.ErrorIs(t, tt.wantErr, err, "DelayedDeleteUsersURLs(%v, %v)", tt.args.userID, tt.args.shortUrls)

			startWaiting := time.Now()
			for {
				time.Sleep(time.Second * 1)
				if mock.ExpectationsWereMet() == nil || time.Since(startWaiting) > time.Second*5 {
					break
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
