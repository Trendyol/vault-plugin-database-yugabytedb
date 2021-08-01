package main

import (
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/database/dbplugin/v5"

	"log"
	"os"
)

func main() {
	apiClientMeta := &api.PluginAPIClientMeta{}
	flags := apiClientMeta.FlagSet()
	flags.Parse(os.Args[1:])

	err := Run()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func Run() error {
	dbType, err := New()
	if err != nil {
		return err
	}

	dbplugin.Serve(dbType.(dbplugin.Database))

	return nil
}
