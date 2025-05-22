package main

import (
	"errors"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/uVazzi/otus_sql_migrator/internal/app/provider"
)

var commands = &cobra.Command{
	Use:  "gomigrator",
	Long: `Simple Database Migration Tool for PostgreSQL`,
}

var ErrIncorrectOrEmptyMigrationName = errors.New("incorrect or empty migration name")

func main() {
	flagDSN, flagDir, flagConfigFilePath := getParams()
	appProvider, closeCallback := provider.NewContainer(flagDSN, flagDir, flagConfigFilePath)
	defer closeCallback()

	addMigratorCommands(appProvider)
	err := commands.Execute()
	if err != nil {
		log.Fatalln(err) //nolint:gocritic
	}
}

func getParams() (string, string, string) {
	var DSN string
	var Dir string
	var configFilePath string
	commands.PersistentFlags().StringVar(&DSN, "dsn", "", "PostgreSQL DSN")
	commands.PersistentFlags().StringVar(&Dir, "dir", "", "Path to migrations directory")
	commands.PersistentFlags().StringVar(&configFilePath, "config", "", "Path to configuration file")

	// Позже cobra сама сделает ParseFlags при Execute, но нам нужны данные до Execute
	_ = commands.ParseFlags(os.Args[1:])

	return DSN, Dir, configFilePath
}

func addMigratorCommands(appProvider *provider.AppContainer) {
	migrator := appProvider.GetMigrator()

	commandCreate := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new migration file",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return ErrIncorrectOrEmptyMigrationName
			}

			err := migrator.Create(args[0])
			if err != nil {
				os.Exit(1)
			}

			return nil
		},
	}

	commandUp := &cobra.Command{
		Use:   "up",
		Short: "Apply all up",
		Run: func(_ *cobra.Command, _ []string) {
			err := migrator.Up()
			if err != nil {
				os.Exit(1)
			}
		},
	}

	commandDown := &cobra.Command{
		Use:   "down",
		Short: "Rollback last migration",
		Run: func(_ *cobra.Command, _ []string) {
			err := migrator.Down()
			if err != nil {
				os.Exit(1)
			}
		},
	}

	commandRedo := &cobra.Command{
		Use:   "redo",
		Short: "Redo last migration",
		Run: func(_ *cobra.Command, _ []string) {
			err := migrator.Redo()
			if err != nil {
				os.Exit(1)
			}
		},
	}

	commandStatus := &cobra.Command{
		Use:   "status",
		Short: "Print the status migrations",
		Run: func(_ *cobra.Command, _ []string) {
			err := migrator.Status()
			if err != nil {
				os.Exit(1)
			}
		},
	}

	commandVersionDB := &cobra.Command{
		Use:   "dbversion",
		Short: "Print the current version database",
		Run: func(_ *cobra.Command, _ []string) {
			err := migrator.VersionDB()
			if err != nil {
				os.Exit(1)
			}
		},
	}

	commands.AddCommand(commandCreate, commandUp, commandDown, commandRedo, commandStatus, commandVersionDB)
}
