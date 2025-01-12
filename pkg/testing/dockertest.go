package testingx

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	connMaxLifeTime = 5 * time.Minute
	expiredSeconds  = 180
	mysqlTag        = "9.0"
)

func NewMysqlInDocker() (*dockertest.Pool, *dockertest.Resource, *gorm.DB, error) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, nil, err
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mysql",
		Tag:        mysqlTag,
		Env:        []string{"MYSQL_ROOT_PASSWORD=secret"},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return nil, nil, nil, err
	}

	if err = resource.Expire(expiredSeconds); err != nil {
		return nil, nil, nil, err
	}

	var db *sql.DB
	var gdb *gorm.DB

	dsn := fmt.Sprintf("root:secret@(localhost:%s)/mysql", resource.GetPort("3306/tcp")) + "?collation=utf8_unicode_ci&parseTime=true&multiStatements=true"

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		gdb, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			return err
		}

		db, err = gdb.DB()
		if err != nil {
			return err
		}

		return db.Ping()
	})
	if err != nil {
		return nil, nil, nil, err
	}

	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(25)
	db.SetConnMaxLifetime(connMaxLifeTime)
	return pool, resource, gdb, nil
}
