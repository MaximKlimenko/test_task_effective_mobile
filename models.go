package main

import "time"

type Song struct {
	ID          int       `db:"id" json:"id"`
	Group       string    `db:"group_name" json:"group"`
	Song        string    `db:"song_name" json:"song"`
	ReleaseDate time.Time `db:"release_date" json:"releaseDate"`
	Lyrics      []string  `db:"lyrics" json:"lyrics"`
	Link        string    `db:"link" json:"link"`
}

type AddSongRequest struct {
	Group string `json:"group" validate:"required"`
	Song  string `json:"song" validate:"required"`
}
