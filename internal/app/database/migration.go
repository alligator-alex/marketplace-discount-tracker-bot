package database

import (
	"bot/internal/app/helpers"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stoewer/go-strcase"
)

const (
	migrationsDir        string = "schema"
	fileNamePrefixFormat string = "2006_01_02_150405"
	fileNameSuffixUp     string = ".up.sql"
	fileNameSuffixDown   string = ".down.sql"
)

// Apply new migrations in a single batch.
func (db *Postgres) Migrate() {
	files := db.getFilesToMigrate()

	if len(files) == 0 {
		fmt.Println("Nothing to migrate.")
		return
	}

	batch := db.getLatestBatch()
	batch += 1

	fmt.Println("Running migrations.")

	for _, fileName := range files {
		if db.checkIfMigrated(fileName) {
			continue
		}

		start := time.Now()
		migrationName := strings.TrimSuffix(fileName, fileNameSuffixUp)

		fmt.Print(migrationName)

		if err := db.migrateOne(fileName, batch); err != nil {
			fmt.Println("Error:", err)
			return
		}

		fmt.Println(helpers.ConcatStrings("..........", time.Since(start).Round(time.Millisecond).String(), " DONE"))
	}
}

// Rollback latest migrations in single batch.
func (db *Postgres) RollbackMigrations() {
	files := db.getFilesToRollback()

	if len(files) == 0 {
		fmt.Println("Nothing to rollback.")
		return
	}

	fmt.Println("Rolling back migrations.")

	for _, fileName := range files {
		start := time.Now()
		migrationName := strings.TrimSuffix(fileName, fileNameSuffixDown)

		fmt.Print(migrationName)

		if err := db.rollbackOne(fileName); err != nil {
			fmt.Println("Error:", err)
			return
		}

		fmt.Println(helpers.ConcatStrings("..........", time.Since(start).Round(time.Millisecond).String(), " DONE"))
	}
}

// Create new migration files.
func CreateNewMigration(name string) {
	var fileName string
	var filePath string

	currentDate := time.Now()

	migrationName := helpers.ConcatStrings(currentDate.Format(fileNamePrefixFormat), "_", strcase.SnakeCase(name))

	fileName = helpers.ConcatStrings(migrationName, fileNameSuffixUp)
	filePath = filepath.Join(getMigrationsDir(), fileName)

	if _, err := os.Create(filePath); err != nil {
		fmt.Println("Unable to create up migration file:", err)
		return
	}

	fmt.Printf("Up migration [%s] created successfully.\n", filepath.Join(migrationsDir, fileName))

	fileName = helpers.ConcatStrings(migrationName, fileNameSuffixDown)
	filePath = filepath.Join(getMigrationsDir(), fileName)

	if _, err := os.Create(filePath); err != nil {
		fmt.Println("Unable to create down migration file:", err)
		return
	}

	fmt.Printf("Down migration [%s] created successfully.\n", filepath.Join(migrationsDir, fileName))
}

// Migrate single file within transaction.
func (db *Postgres) migrateOne(fileName string, batch int) error {
	filePath := filepath.Join(getMigrationsDir(), fileName)
	fileContent, err := os.ReadFile(filePath)

	if err != nil {
		return err
	}

	transaction, err := database.Connection.Begin(database.Context)

	if err != nil {
		return err
	}

	defer transaction.Rollback(database.Context)

	sql := strings.TrimSpace(string(fileContent))

	if _, err := database.Connection.Exec(database.Context, sql); err != nil {
		return err
	}

	if err := db.addMigration(fileName, batch); err != nil {
		return err
	}

	return transaction.Commit(database.Context)
}

// Rollback single file within transaction.
func (db *Postgres) rollbackOne(fileName string) error {
	filePath := filepath.Join(getMigrationsDir(), fileName)
	fileContent, err := os.ReadFile(filePath)

	if err != nil {
		return err
	}

	transaction, err := database.Connection.Begin(database.Context)

	if err != nil {
		return err
	}

	defer transaction.Rollback(database.Context)

	sql := strings.TrimSpace(string(fileContent))

	if _, err := database.Connection.Exec(database.Context, sql); err != nil {
		return err
	}

	if err := db.deleteMigration(fileName); err != nil {
		return err
	}

	return transaction.Commit(database.Context)
}

// Get migrations files directory.
func getMigrationsDir() string {
	rootDir, err := helpers.GetRootDir()

	if err != nil {
		fmt.Println("Unable to get root directory:", err)
		os.Exit(1)
	}

	return filepath.Join(rootDir, "schema")
}

// Get list of files that must be migrated.
func (db *Postgres) getFilesToMigrate() []string {
	migrationsDir := getMigrationsDir()
	filesList, err := os.ReadDir(migrationsDir)

	if err != nil {
		fmt.Println("Unable to read migration files:", err)
		os.Exit(1)
	}

	var migrations []string

	for _, file := range filesList {
		fileName := file.Name()

		if !strings.HasSuffix(fileName, fileNameSuffixUp) {
			continue
		}

		if db.checkIfMigrated(fileName) {
			continue
		}

		migrations = append(migrations, fileName)
	}

	return migrations
}

// Get files list to rollback.
func (db *Postgres) getFilesToRollback() []string {
	latestBatch := db.getLatestBatch()

	sql := "select migration, batch from migrations order by id desc;"

	rows, _ := database.Connection.Query(database.Context, sql)

	defer rows.Close()

	var migrations []string

	for rows.Next() {
		var migrationName string
		var batch int

		rows.Scan(&migrationName, &batch)

		if batch < latestBatch {
			break
		}

		fileName := helpers.ConcatStrings(migrationName, fileNameSuffixDown)

		migrations = append(migrations, fileName)
	}

	return migrations
}

// Insert successfull migration into migrations table.
func (db *Postgres) addMigration(fileName string, batch int) error {
	migrationName := strings.TrimSuffix(fileName, fileNameSuffixUp)

	sql := "INSERT INTO migrations (migration, batch) VALUES ($1, $2);"

	_, err := database.Connection.Exec(database.Context, sql, migrationName, batch)

	return err
}

// Delete rolled back migration from migrations table.
func (db *Postgres) deleteMigration(fileName string) error {
	migrationName := strings.TrimSuffix(fileName, fileNameSuffixDown)

	sql := "DELETE FROM migrations WHERE migration = $1;"

	_, err := database.Connection.Exec(database.Context, sql, migrationName)

	return err
}

// Check if migration has been applied.
func (db *Postgres) checkIfMigrated(fileName string) bool {
	migrationName := strings.TrimSuffix(fileName, fileNameSuffixUp)

	var isMigrated bool

	sql := "SELECT EXISTS(SELECT 1 FROM migrations WHERE migration = $1);"
	err := database.Connection.QueryRow(database.Context, sql, migrationName).Scan(&isMigrated)

	if err == pgx.ErrNoRows {
		return false
	}

	if err != nil {
		fmt.Println("Unable to check if migrated:", err)
		os.Exit(1)
	}

	return isMigrated
}

// Get latest migration batch number.
func (db *Postgres) getLatestBatch() int {
	var latestBatch int

	sql := "SELECT batch FROM migrations ORDER BY id DESC LIMIT 1;"
	err := database.Connection.QueryRow(database.Context, sql).Scan(&latestBatch)

	if err == pgx.ErrNoRows {
		return 0
	}

	if err != nil {
		fmt.Println("Unable to fetch latest migration batch:", err)
		os.Exit(1)
	}

	return latestBatch
}
