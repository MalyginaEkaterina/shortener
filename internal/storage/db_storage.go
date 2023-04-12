package storage

import (
	"context"
	"database/sql"
	"errors"
	"github.com/MalyginaEkaterina/shortener/internal"
	"log"
	"time"
)

var _ Storage = (*DBStorage)(nil)

// DBStorage contains *sql.DB and prepared statements.
type DBStorage struct {
	DB               *sql.DB
	insertUser       *sql.Stmt
	insertURL        *sql.Stmt
	selectURLByID    *sql.Stmt
	selectUrlsByUser *sql.Stmt
	selectURLID      *sql.Stmt
	deleteURL        *sql.Stmt
}

// NewDBStorage opens sql connection, prepares statements and returns *DBStorage.
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
	stmtInsertURL, err := db.Prepare("INSERT INTO urls (original_url, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING RETURNING id")
	if err != nil {
		return nil, err
	}
	stmtSelectURLByID, err := db.Prepare("SELECT original_url, is_deleted FROM urls WHERE id = $1")
	if err != nil {
		return nil, err
	}
	stmtSelectUrlsByUser, err := db.Prepare("SELECT id, original_url FROM urls WHERE user_id = $1")
	if err != nil {
		return nil, err
	}
	stmtSelectURLID, err := db.Prepare("SELECT id FROM urls WHERE original_url = $1")
	if err != nil {
		return nil, err
	}
	stmtDeleteURL, err := db.Prepare("UPDATE urls SET is_deleted = true WHERE id = $1 AND user_id = $2")
	if err != nil {
		return nil, err
	}
	return &DBStorage{
		DB:               db,
		insertUser:       stmtInsertUser,
		insertURL:        stmtInsertURL,
		selectURLByID:    stmtSelectURLByID,
		selectUrlsByUser: stmtSelectUrlsByUser,
		selectURLID:      stmtSelectURLID,
		deleteURL:        stmtDeleteURL,
	}, nil
}

func initTables(db *sql.DB) error {
	const createUsersTableSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id serial PRIMARY KEY
		)
	`
	const createUrlsTableSQL = `
		CREATE TABLE IF NOT EXISTS urls (
			id bigserial PRIMARY KEY,
			original_url varchar,
			user_id integer,
			is_deleted boolean DEFAULT false,
			UNIQUE(original_url),
			FOREIGN KEY (user_id) REFERENCES users (id)
	   )
	`
	_, err := db.Exec(createUsersTableSQL)
	if err != nil {
		return err
	}
	_, err = db.Exec(createUrlsTableSQL)
	if err != nil {
		return err
	}
	return nil
}

// AddUser inserts new user and returns its id.
func (d DBStorage) AddUser(ctx context.Context) (int, error) {
	row := d.insertUser.QueryRowContext(ctx)
	var id int
	err := row.Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// AddURL inserts new URL and returns its id or ErrAlreadyExists if this URL already exists.
func (d DBStorage) AddURL(ctx context.Context, url string, userID int) (int, error) {
	row := d.insertURL.QueryRowContext(ctx, url, userID)
	var id int
	err := row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrAlreadyExists
	} else if err != nil {
		return 0, err
	}
	return id, nil
}

// GetURLID returns url id by its url string.
func (d DBStorage) GetURLID(ctx context.Context, url string) (int, error) {
	row := d.selectURLID.QueryRowContext(ctx, url)
	var id int
	err := row.Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetURL returns URL by its id. Returns ErrNotFound if there is no such id or ErrDeleted if id is marked as deleted.
func (d DBStorage) GetURL(ctx context.Context, id string) (string, error) {
	row := d.selectURLByID.QueryRowContext(ctx, id)
	var originalURL string
	var isDeleted bool
	err := row.Scan(&originalURL, &isDeleted)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	} else if err != nil {
		return "", err
	}
	if isDeleted {
		return "", ErrDeleted
	}
	return originalURL, nil
}

// GetUserUrls returns a map with ids and their original URLs for all URLs for the user.
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

// AddBatch inserts the list of URLs in one transaction.
func (d DBStorage) AddBatch(ctx context.Context, urls []internal.CorrIDOriginalURL, userID int) ([]internal.CorrIDUrlID, error) {
	tx, err := d.DB.Begin()
	if err != nil {
		log.Println("Begin transaction error", err)
		return nil, err
	}
	txStmt := tx.StmtContext(ctx, d.insertURL)
	defer txStmt.Close()
	var corrURLIDs []internal.CorrIDUrlID
	for _, v := range urls {
		row := txStmt.QueryRowContext(ctx, v.OriginalURL, userID)
		corrURLID := internal.CorrIDUrlID{CorrID: v.CorrID}
		err = row.Scan(&corrURLID.URLID)
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

// DeleteBatch marks URLs by ids from the list as deleted in one transaction.
func (d DBStorage) DeleteBatch(ctx context.Context, ids []internal.IDToDelete) error {
	tx, err := d.DB.Begin()
	if err != nil {
		log.Println("Begin transaction error", err)
		return err
	}
	defer tx.Rollback()

	txStmt := tx.StmtContext(ctx, d.deleteURL)
	defer txStmt.Close()
	for _, v := range ids {
		_, err = txStmt.ExecContext(ctx, v.ID, v.UserID)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Println("Commit error", err)
		return err
	}
	return nil
}

// Close closes prepared statements and sql connection.
func (d DBStorage) Close() {
	d.insertUser.Close()
	d.insertURL.Close()
	d.selectURLByID.Close()
	d.selectUrlsByUser.Close()
	d.selectURLID.Close()
	d.deleteURL.Close()
	d.DB.Close()
}

// Ping check the sql connection.
func (d DBStorage) Ping(c context.Context) error {
	ctx, cancel := context.WithTimeout(c, 1*time.Second)
	defer cancel()
	return d.DB.PingContext(ctx)
}
