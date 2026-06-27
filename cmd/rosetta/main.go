package main

import (
	"github.com/asabla/rosetta/internal/api"
	"github.com/asabla/rosetta/internal/authz"
	"github.com/asabla/rosetta/internal/openshell"
	"github.com/asabla/rosetta/internal/store"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	st, err := store.Open("rosetta.db")
	if err != nil {
		panic(err)
	}
	srv := api.Server{Store: st, Auth: authz.NewCedarAuthorizer(log), Shell: openshell.LoggingAdapter{Log: log}, Dir: "generated-policies"}
	log.Info("listening", "addr", ":8080")
	panic(http.ListenAndServe(":8080", srv.Routes()))
}
