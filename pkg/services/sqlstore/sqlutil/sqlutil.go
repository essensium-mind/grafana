package sqlutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ITestDB is an interface of arguments for testing db
type ITestDB interface {
	Helper()
	Fatalf(format string, args ...any)
	Logf(format string, args ...any)
	Log(args ...any)
	Cleanup(func())
}

type TestDB struct {
	DriverName string
	ConnStr    string
	Path       string
}

func GetTestDBType() string {
	dbType := "sqlite3"

	// environment variable present for test db?
	if db, present := os.LookupEnv("GRAFANA_TEST_DB"); present {
		dbType = db
	}
	return dbType
}

func GetTestDB(t ITestDB, dbType string) (*TestDB, error) {
	switch dbType {
	case "mysql":
		return MySQLTestDB()
	case "postgres":
		return PostgresTestDB()
	case "sqlite3":
		return SQLite3TestDB()
	}

	return nil, fmt.Errorf("unknown test db type: %s", dbType)
}

func SQLite3TestDB() (*TestDB, error) {
	if os.Getenv("SQLITE_INMEMORY") == "true" {
		return &TestDB{
			DriverName: "sqlite3",
			ConnStr:    "file::memory:",
		}, nil
	}

	sqliteDb := os.Getenv("SQLITE_TEST_DB")
	if sqliteDb == "" {
		// try to create a database file in the user's cache directory
		dir, err := os.UserCacheDir()
		if err != nil {
			return nil, err
		}

		// if cache dir doesn't exist, fall back to temp dir
		if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
			dir = os.TempDir()
			if _, err := os.Stat(dir); err != nil {
				return nil, err
			}
		}

		err = os.Mkdir(filepath.Join(dir, "grafana-test"), 0750)
		if err != nil && !errors.Is(err, fs.ErrExist) {
			return nil, err
		}

		sqliteDb = filepath.Join(dir, "grafana-test", "grafana-test.db")
	}

	// remove db file if it exists
	err := os.Remove(sqliteDb)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	// remove wal & shm files if they exist
	err = os.Remove(sqliteDb + "-wal")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	err = os.Remove(sqliteDb + "-shm")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	//#nosec G304 - this is a test db
	f, err := os.Create(sqliteDb)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}

	connstr := "file:" + f.Name() + "?cache=private&mode=rwc"
	if os.Getenv("SQLITE_JOURNAL_MODE") != "false" {
		connstr = connstr + "&_journal_mode=WAL"
	}

	return &TestDB{
		DriverName: "sqlite3",
		ConnStr:    connstr,
		Path:       f.Name(),
	}, nil
}

func MySQLTestDB() (*TestDB, error) {
	host := os.Getenv("MYSQL_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("MYSQL_PORT")
	if port == "" {
		port = "3306"
	}
	conn_str := fmt.Sprintf("grafana:password@tcp(%s:%s)/grafana_tests?collation=utf8mb4_unicode_ci&sql_mode='ANSI_QUOTES'&parseTime=true", host, port)
	return &TestDB{
		DriverName: "mysql",
		ConnStr:    conn_str,
	}, nil
}

func PostgresTestDB() (*TestDB, error) {
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}
	connStr := fmt.Sprintf("user=grafanatest password=grafanatest host=%s port=%s dbname=grafanatest sslmode=disable", host, port)
	return &TestDB{
		DriverName: "postgres",
		ConnStr:    connStr,
	}, nil
}
