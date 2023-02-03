package storage

import (
	"context"
	"database/sql"
	"github.com/MalyginaEkaterina/shortener/internal"
	"log"
	"time"
)

type DBStorage struct {
	DB               *sql.DB
	insertUser       *sql.Stmt
	insertURL        *sql.Stmt
	selectURLByID    *sql.Stmt
	selectUrlsByUser *sql.Stmt
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
	stmtInsertUser, err := db.Prepare("INSERT INTO users DEFAULT VALUES RETURNING id")
	if err != nil {
		return nil, err
	}
	stmtInsertURL, err := db.Prepare("INSERT INTO urls (original_url, user_id) VALUES ($1, $2) RETURNING id")
	if err != nil {
		return nil, err
	}
	stmtSelectURLByID, err := db.Prepare("SELECT original_url FROM urls WHERE id = $1")
	if err != nil {
		return nil, err
	}
	stmtSelectUrlsByUser, err := db.Prepare("SELECT id, original_url FROM urls WHERE user_id = $1")
	if err != nil {
		return nil, err
	}
	return &DBStorage{
		DB:               db,
		insertUser:       stmtInsertUser,
		insertURL:        stmtInsertURL,
		selectURLByID:    stmtSelectURLByID,
		selectUrlsByUser: stmtSelectUrlsByUser,
	}, nil
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
	row := d.insertUser.QueryRowContext(ctx)
	var id int
	err := row.Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (d DBStorage) AddURL(ctx context.Context, url string, userID int) (int, error) {
	row := d.insertURL.QueryRowContext(ctx, url, userID)
	var id int
	err := row.Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (d DBStorage) GetURL(ctx context.Context, id string) (string, error) {
	row := d.selectURLByID.QueryRowContext(ctx, id)
	var originalURL string
	err := row.Scan(&originalURL)
	if err != nil {
		return "", err
	}
	return originalURL, nil
}

func (d DBStorage) GetUserUrls(ctx context.Context, userID int) (map[int]string, error) {
	rows, err := d.selectUrlsByUser.QueryContext(ctx, userID)
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

func (d DBStorage) AddBatch(ctx context.Context, urls []internal.CorrIDOriginalURL, userID int) ([]internal.CorrIDUrlID, error) {
	tx, err := d.DB.Begin()
	if err != nil {
		log.Println("Begin transaction error", err)
		return nil, err
	}
	txStmt := tx.StmtContext(ctx, d.insertURL)
	var corrURLIDs []internal.CorrIDUrlID
	for _, v := range urls {
		row := txStmt.QueryRowContext(ctx, v.OriginalURL, userID)
		corrURLID := internal.CorrIDUrlID{CorrID: v.CorrID}
		err := row.Scan(&corrURLID.URLID)
		if err == nil {
			corrURLIDs = append(corrURLIDs, corrURLID)
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Println("Commit error", err)
		return nil, err
	}
	return corrURLIDs, nil
}

func (d DBStorage) Close() {
	d.insertUser.Close()
	d.insertURL.Close()
	d.selectURLByID.Close()
	d.selectUrlsByUser.Close()
	d.DB.Close()
}

func (d DBStorage) Ping(c context.Context) error {
	ctx, cancel := context.WithTimeout(c, 1*time.Second)
	defer cancel()
	return d.DB.PingContext(ctx)
}
