package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/hashicorp/vault/sdk/database/dbplugin/v5"
	"github.com/hashicorp/vault/sdk/database/helper/dbutil"
	"github.com/hashicorp/vault/sdk/helper/dbtxn"
	"github.com/hashicorp/vault/sdk/helper/strutil"
	"github.com/hashicorp/vault/sdk/helper/template"
	"log"
	"strings"
	"time"
)

const (
	yugabyteDBType = "yugabyte"

	defaultUserNameTemplate = `V_{{.DisplayName | uppercase | truncate 64}}_{{.RoleName | uppercase | truncate 64}}_{{random 20 | uppercase}}_{{unix_time}}`
)

type YugabyteDB struct {
	SQLConnectionProducer
	usernameProducer template.StringTemplate
}

func New() (interface{}, error) {
	db := newYugabyteDB()

	// This middleware isn't strictly required, but highly recommended to prevent accidentally exposing
	// values such as passwords in error messages. An example of this is included below
	dbType := dbplugin.NewDatabaseErrorSanitizerMiddleware(db, db.secretValues)
	return dbType, nil
}

var _ dbplugin.Database = (*YugabyteDB)(nil)

func newYugabyteDB() *YugabyteDB {
	connProducer := SQLConnectionProducer{}
	connProducer.Type = yugabyteDBType

	yugabyte := &YugabyteDB{
		SQLConnectionProducer: connProducer,
	}
	return yugabyte
}

func (db *YugabyteDB) secretValues() map[string]string {
	return map[string]string{
		db.Password: "[password]",
		db.Username: "[username]",
	}
}

func (db *YugabyteDB) Initialize(ctx context.Context, req dbplugin.InitializeRequest) (dbplugin.InitializeResponse, error) {
	usernameTemplate, err := strutil.GetString(req.Config, "username_template")
	if err != nil {
		return dbplugin.InitializeResponse{}, fmt.Errorf("failed to retrieve username_template: %w", err)
	}

	log.Println("initializing --> ", usernameTemplate)

	if usernameTemplate == "" {
		usernameTemplate = defaultUserNameTemplate
	}

	up, err := template.NewTemplate(template.Template(usernameTemplate))
	if err != nil {
		return dbplugin.InitializeResponse{}, fmt.Errorf("unable to initialize username template: %w", err)
	}
	db.usernameProducer = up

	_, err = db.usernameProducer.Generate(dbplugin.UsernameMetadata{})
	if err != nil {
		return dbplugin.InitializeResponse{}, fmt.Errorf("invalid username template: %w", err)
	}

	err = db.SQLConnectionProducer.Initialize(ctx, req.Config, req.VerifyConnection)
	if err != nil {
		return dbplugin.InitializeResponse{}, err
	}
	resp := dbplugin.InitializeResponse{
		Config: req.Config,
	}
	return resp, nil
}

func (db *YugabyteDB) NewUser(ctx context.Context, req dbplugin.NewUserRequest) (dbplugin.NewUserResponse, error) {
	statements := removeEmpty(req.Statements.Commands)
	if len(statements) == 0 {
		return dbplugin.NewUserResponse{}, dbutil.ErrEmptyCreationStatement
	}

	db.Lock()
	defer db.Unlock()

	username, err := db.usernameProducer.Generate(req.UsernameConfig)
	if err != nil {
		return dbplugin.NewUserResponse{}, fmt.Errorf("failed to generate username: %w", err)
	}

	conn, err := db.getConnection(ctx)
	if err != nil {
		return dbplugin.NewUserResponse{}, fmt.Errorf("failed to get connection: %w", err)
	}

	err = newUser(ctx, conn, username, req.Password, req.Expiration, req.Statements.Commands)
	if err != nil {
		return dbplugin.NewUserResponse{}, err
	}

	resp := dbplugin.NewUserResponse{
		Username: username,
	}
	return resp, nil
}

func removeEmpty(strs []string) []string {
	newStrs := []string{}
	for _, str := range strs {
		str = strings.TrimSpace(str)
		if str == "" {
			continue
		}
		newStrs = append(newStrs, str)
	}
	return newStrs
}

func newUser(ctx context.Context, db *sql.DB, username, password string, expiration time.Time, commands []string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start a transaction: %w", err)
	}
	// Effectively a no-op if the transaction commits successfully
	defer tx.Rollback()

	for _, stmt := range commands {
		for _, query := range strutil.ParseArbitraryStringSlice(stmt, ";") {
			query = strings.TrimSpace(query)
			if len(query) == 0 {
				continue
			}

			m := map[string]string{
				"username":   username,
				"name":       username, // backwards compatibility
				"password":   password,
				"expiration": expiration.Format("02-01-2006 15:04:05 PM"),
			}

			err = dbtxn.ExecuteTxQuery(ctx, tx, m, query)
			if err != nil {
				return fmt.Errorf("failed to execute query: %w, query is :%s, m:%v", err, query, m)
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (db *YugabyteDB) UpdateUser(ctx context.Context, req dbplugin.UpdateUserRequest) (dbplugin.UpdateUserResponse, error) {
	panic("implement me")
}

func (db *YugabyteDB) DeleteUser(ctx context.Context, req dbplugin.DeleteUserRequest) (dbplugin.DeleteUserResponse, error) {
	panic("implement me")
}

func (db *YugabyteDB) Type() (string, error) {
	return yugabyteDBType, nil
}

func (db *YugabyteDB) Close() error {
	panic("implement me")
}

func (db *YugabyteDB) getConnection(ctx context.Context) (*sql.DB, error) {
	conn, err := db.Connection(ctx)
	if err != nil {
		return nil, err
	}

	return conn.(*sql.DB), nil
}
