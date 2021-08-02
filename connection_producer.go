package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/pkg/errors"
	"sync"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/mitchellh/mapstructure"

	_ "github.com/lib/pq"
)

// SQLConnectionProducer implements ConnectionProducer and provides a generic producer for most sql databases
type SQLConnectionProducer struct {
	Host     string `json:"host" mapstructure:"host" structs:"host"`
	Username string `json:"username" mapstructure:"username" structs:"username"`
	Password string `json:"password" mapstructure:"password" structs:"password"`
	Port     int    `json:"port" mapstructure:"port" structs:"port"`
	DbName   string `json:"db" mapstructure:"db" structs:"db"`

	RawConfig             map[string]interface{}
	maxConnectionLifetime time.Duration
	Initialized           bool
	db                    *sql.DB
	sync.Mutex
	Type string
}

func (c *SQLConnectionProducer) Initialize(ctx context.Context, conf map[string]interface{}, verifyConnection bool) error {
	_, err := c.Init(ctx, conf, verifyConnection)
	return err
}

func (c *SQLConnectionProducer) Init(ctx context.Context, conf map[string]interface{}, verifyConnection bool) (map[string]interface{}, error) {
	c.Lock()
	defer c.Unlock()

	c.RawConfig = conf

	decoderConfig := &mapstructure.DecoderConfig{
		Result:           c,
		WeaklyTypedInput: true,
		TagName:          "json",
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, err
	}

	err = decoder.Decode(conf)
	if err != nil {
		return nil, err
	}

	switch {
	case len(c.Host) == 0:
		return nil, fmt.Errorf("host cannot be empty")
	case len(c.Username) == 0:
		return nil, fmt.Errorf("username cannot be empty")
	case len(c.Password) == 0:
		return nil, fmt.Errorf("password cannot be empty")
	}

	// Set initialized to true at this point since all fields are set,
	// and the connection can be established at a later time.
	c.Initialized = true

	if verifyConnection {
		if _, err := c.Connection(ctx); err != nil {
			return nil, errwrap.Wrapf("error verifying connection: {{err}}", err)
		}

		if err := c.db.PingContext(ctx); err != nil {
			return nil, errwrap.Wrapf("error verifying connection: {{err}}", err)
		}
	}

	return c.RawConfig, nil
}

func (c *SQLConnectionProducer) Connection(ctx context.Context) (interface{}, error) {
	if !c.Initialized {
		return nil, errors.New("not initialized")
	}

	// If we already have a DB, test it and return
	if c.db != nil {
		if err := c.db.PingContext(ctx); err == nil {
			return c.db, nil
		}
		// If the ping was unsuccessful, close it and ignore errors as we'll be
		// reestablishing anyways
		c.db.Close()
	}

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		c.Host, c.Port, c.Username, c.Password, c.DbName)

	var err error
	c.db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	return c.db, nil
}

// Close attempts to close the connection
func (c *SQLConnectionProducer) Close() error {
	// Grab the write lock
	c.Lock()
	defer c.Unlock()

	if c.db != nil {
		c.db.Close()
	}

	c.db = nil

	return nil
}
