package main

import (
	"context"
	"fmt"
	"github.com/hashicorp/vault/sdk/database/dbplugin/v5"
	"github.com/hashicorp/vault/sdk/helper/strutil"
	"github.com/hashicorp/vault/sdk/helper/template"
	"log"
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
	panic("implement me")
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
