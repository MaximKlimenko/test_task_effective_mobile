package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

func setupRoutes(r *chi.Mux, logger *logrus.Logger) {
	r.Get("/library", getLibrary)
	r.Post("/add", addSong)
	r.Get("/lyrics", getSongLyrics)
	r.Delete("/delete/{id}", deleteSong)
	r.Put("/update/{id}", updateSong)
}
