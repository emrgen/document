package tester

import (
	"github.com/ory/dockertest/v3"
	"github.com/sirupsen/logrus"
)

func SetupDocker() (func(), error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		logrus.Fatalf("Could not construct pool: %s", err)
	}

	// uses pool to try to connect to Docker
	err = pool.Client.Ping()
	if err != nil {
		logrus.Fatalf("Could not connect to Docker: %s", err)
	}

	// run database
	db, err := pool.Run("postgres", "9.6", []string{
		"POSTGRES_USER=emrgen",
		"POSTGRES_PASSWORD=emrgen",
		"POSTGRES_DB=emrgen",
	})
	if err != nil {
		logrus.Fatalf("Could not start resource: %s", err)
	}

	// run spicedb migrate
	migrate, err := pool.Run("spicedb", "latest", []string{
		"DATABASE_URL=postgres://emrgen:emrgen@localhost:5432/emrgen?sslmode=disable",
	})
	if err != nil {
		logrus.Fatalf("Could not start resource: %s", err)
	}

	code, err := migrate.Exec([]string{"spicedb", "migrate"}, dockertest.ExecOptions{})
	if err != nil {
		logrus.Fatalf("Could not run command: %s", err)
	}

	if code != 0 {
		logrus.Fatalf("Could not run command: %d", code)
	}

	pool.Purge(migrate)

	// run spicedb
	spicedb, err := pool.RunWithOptions(&dockertest.RunOptions{
		Env: []string{
			"SPICEDB_DATASTORE_ENGINE=postgres",
			"SPICEDB_DATASTORE_CONN_URI=postgres://emrgen:emrgen@localhost:5432/emrgen?sslmode=disable",
			"SPICEDB_GRPC_PRESHARED_KEY=emrgen",
		},
		Repository: "spicedb",
		Tag:        "latest",
		ExposedPorts: []string{
			"50051",
			"8080",
			"9090",
		},
	})
	if err != nil {
		logrus.Fatalf("Could not start resource: %s", err)
	}

	// run authbac
	authbac, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "authbac",
		Tag:        "latest",
		ExposedPorts: []string{
			"4010:4000",
			"4011:4001",
		},
	})
	if err != nil {
		logrus.Fatalf("Could not start resource: %s", err)
	}

	purge := func() {
		if err := pool.Purge(db); err != nil {
			logrus.Fatalf("Could not purge resource: %s", err)
		}

		if err := pool.Purge(authbac); err != nil {
			logrus.Fatalf("Could not purge resource: %s", err)
		}

		if err := pool.Purge(spicedb); err != nil {
			logrus.Fatalf("Could not purge resource: %s", err)
		}

	}

	return purge, nil
}
