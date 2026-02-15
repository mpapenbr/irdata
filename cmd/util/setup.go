package util

import (
	"github.com/dgraph-io/badger/v4"
	"github.com/mpapenbr/irdata/auth"
	"github.com/mpapenbr/irdata/cache"
	"github.com/mpapenbr/irdata/cmd/config"
	"github.com/mpapenbr/irdata/irdata"
	"github.com/mpapenbr/irdata/log"
)

type (
	App struct {
		API *irdata.IrData
		DB  *badger.DB
	}
)

func InitApp() (*App, error) {
	tm, tmErr := auth.NewTokenManager(auth.WithAuthConfig(&config.IrAuthConfig))
	if tmErr != nil {
		log.Error("failed to create token manager", log.ErrorField(tmErr))
		return nil, tmErr
	}
	if loginErr := tm.Login(); loginErr != nil {
		log.Error("failed to login", log.ErrorField(loginErr))
		return nil, loginErr
	}
	db, dbErr := badger.Open(badger.DefaultOptions(config.CacheDir))
	if dbErr != nil {
		log.Error("failed to open cache database", log.ErrorField(dbErr))
		return nil, dbErr
	}

	badgerCache, cacheErr := cache.NewBadgerCache(db)
	if cacheErr != nil {
		log.Error("failed to create cache", log.ErrorField(cacheErr))
		return nil, cacheErr
	}
	ir, irErr := irdata.NewIrData(
		irdata.WithTokenProvider(tm.GetAccessToken),
		irdata.WithCache(badgerCache),
	)
	if irErr != nil {
		log.Error("failed to create iRData instance", log.ErrorField(irErr))
		return nil, irErr
	}
	return &App{API: ir, DB: db}, nil
}

func (a *App) Close() {
	if err := a.DB.Close(); err != nil {
		log.Error("failed to close cache database", log.ErrorField(err))
	}
}
