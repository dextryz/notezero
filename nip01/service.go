package nip01

import (
	"context"
	"log/slog"

	"github.com/dextryz/tenet"
	"github.com/dextryz/tenet/sqlite"
)

type Service struct {
	Log *slog.Logger
	Db  *sqlite.Db
}

func New(l *slog.Logger, d *sqlite.Db) Service {
	return Service{
		Log: l,
		Db:  d,
	}
}

func (s Service) Request(ctx context.Context, pubkey string) ([]*tenet.Profile, error) {

	return nil, nil
}
