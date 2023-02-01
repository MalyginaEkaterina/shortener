package storage

import (
	"context"
	"database/sql"
	"log"
	"time"
)

type DBStorage struct {
	DB *sql.DB
}

var _ Storage = (*DBStorage)(nil)

func NewDBStorage(dsn string) (*DBStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	err = initTables(db)
	if err != nil {
		return nil, err
	}
	return &DBStorage{DB: db}, nil
}

func initTables(db *sql.DB) error {
	userSQL := "CREATE TABLE IF NOT EXISTS users (id serial PRIMARY KEY)"
	urlsSQL := "CREATE TABLE IF NOT EXISTS urls (id bigserial PRIMARY KEY, original_url varchar, user_id integer, FOREIGN KEY (user_id) REFERENCES users (id))"
	_, err := db.Exec(userSQL)
	if err != nil {
		return err
	}
	_, err = db.Exec(urlsSQL)
	if err != nil {
		return err
	}
	return nil
}

func (d DBStorage) AddUser(ctx context.Context) (int, error) {
	insertUser := "INSERT INTO users DEFAULT VALUES RETURNING id"
	row := d.DB.QueryRowContext(ctx, insertUser)
	var id int
	err := row.Scan(&id)
	if err != nil {
		return 0, err
	}
	log.Printf("inserted user id %d\n", id)
	return id, nil
}

func (d DBStorage) AddURL(ctx context.Context, url string, userID int) (int, error) {
	insertURL := "INSERT INTO urls (original_url, user_id) VALUES ($1, $2) RETURNING id"
	row := d.DB.QueryRowContext(ctx, insertURL, url, userID)
	var id int
	err := row.Scan(&id)
	if err != nil {
		return 0, err
	}
	log.Printf("inserted url id %d\n", id)
	return id, nil
}

func (d DBStorage) GetURL(ctx context.Context, id string) (string, error) {
	var originalURL string
	selectURL := "SELECT original_url FROM urls WHERE id = $1"
	row := d.DB.QueryRowContext(ctx, selectURL, id)
	err := row.Scan(&originalURL)
	if err != nil {
		return "", err
	}
	return originalURL, nil
}

func (d DBStorage) GetUserUrls(ctx context.Context, userID int) (map[int]string, error) {
	selectUserUrls := "SELECT id, original_url FROM urls WHERE user_id = $1"
	rows, err := d.DB.QueryContext(ctx, selectUserUrls, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	userUrls := make(map[int]string)

	for rows.Next() {
		var id int
		var originalURL string
		err = rows.Scan(&id, &originalURL)
		if err != nil {
			return nil, err
		}
		userUrls[id] = originalURL
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return userUrls, nil
}

func (d DBStorage) Close() {
	d.DB.Close()
}

func (d DBStorage) Ping(c context.Context) error {
	ctx, cancel := context.WithTimeout(c, 1*time.Second)
	defer cancel()
	return d.DB.PingContext(ctx)
}
