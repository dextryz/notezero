package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/dextryz/tenet"

	"github.com/dextryz/nostr"

	_ "github.com/mattn/go-sqlite3"
)

var ErrDupEvent = errors.New("duplicate: event already exists")
var ErrDupProfile = errors.New("duplicate: profile already exists")

type Db struct {
	*sql.DB
	QueryLimit       int
	QueryIdLimit     int
	QueryAuthorLimit int
	QueryTagLimit    int
}

func (s *Db) Close() {
	s.DB.Close()
}

func createTables(db *sql.DB) error {

	createProfileSQL := `
    CREATE TABLE IF NOT EXISTS profile (
        pubkey TEXT PRIMARY KEY,
        name TEXT,
        about TEXT,
        website TEXT,
        banner TEXT,
        picture TEXT,
        identifier TEXT
    );`

	_, err := db.Exec(createProfileSQL)
	if err != nil {
		return err
	}

	log.Println("table events created")

	return nil
}

func New(database string) *Db {

	db, err := sql.Open("sqlite3", database)
	if err != nil {
		log.Fatal(err)
	}

	err = createTables(db)
	if err != nil {
		log.Fatal(err)
	}

	return &Db{
		DB:               db,
		QueryLimit:       500,
		QueryIdLimit:     10,
		QueryAuthorLimit: 10,
		QueryTagLimit:    10,
	}
}

func (s *Db) StoreProfile(ctx context.Context, p *nostr.Profile, pubkey string) (*tenet.Profile, error) {

	// TODO: Add convertion to npub maybe

	profile := &tenet.Profile{
		PubKey:     pubkey,
		Name:       p.Name,
		About:      p.About,
		Website:    p.Website,
		Banner:     p.Banner,
		Picture:    p.Picture,
		Identifier: p.Nip05,
	}

	err := s.InsertProfile(ctx, profile)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *Db) InsertProfile(ctx context.Context, p *tenet.Profile) error {

	eventSql := "INSERT OR IGNORE INTO profile (pubkey, name, about, website, banner, picture, identifier) VALUES ($1, $2, $3, $4, $5, $6, $7)"

	res, err := s.DB.ExecContext(ctx, eventSql, p.PubKey, p.Name, p.About, p.Website, p.Banner, p.Picture, p.Identifier)
	if err != nil {
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		return err
	}

	return nil
}

func (s *Db) QueryProfileByPubkey(pubkey string) (*tenet.Profile, error) {

	rows := s.DB.QueryRow(`SELECT * FROM profile WHERE pubkey = ?`, pubkey)

	var p tenet.Profile
	err := rows.Scan(&p.PubKey, &p.Name, &p.About, &p.Website, &p.Banner, &p.Picture, &p.Identifier)
	if err != nil {
		return nil, err
	}

	return &p, nil
}
