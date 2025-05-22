package provider

import (
	"database/sql"
	"log"

	"github.com/uVazzi/otus_sql_migrator/internal/config"
	"github.com/uVazzi/otus_sql_migrator/pkg/gomigrator"
)

type AppContainer struct {
	conf     *config.Config
	db       *sql.DB
	migrator gomigrator.Migrator
}

func NewContainer(flagDSN, flagDir, flagConfigFilePath string) (*AppContainer, func()) {
	conf, err := config.NewConfig(flagDSN, flagDir, flagConfigFilePath)
	if err != nil {
		log.Fatalln("Fatal error on create config: " + err.Error())
	}

	db, err := newDB(conf.DSN)
	if err != nil {
		log.Fatalln("Fatal error on init db: " + err.Error())
	}

	migrator := gomigrator.NewMigrator(db, conf.Dir)

	appContainer := &AppContainer{
		conf:     conf,
		db:       db,
		migrator: migrator,
	}

	closeCallback := func() {
		if appContainer.db.Stats().Idle != 0 || appContainer.db.Stats().InUse != 0 {
			err := appContainer.db.Close()
			if err != nil {
				log.Fatalln("Fatal error on close db: " + err.Error())
			}
		}
	}

	return appContainer, closeCallback
}

func (container *AppContainer) GetMigrator() gomigrator.Migrator {
	return container.migrator
}
