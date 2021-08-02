#  vault-database-plugin-yugabytedb
This project aims to implement [Custom Database plugin interface](https://www.vaultproject.io/docs/secrets/databases/custom) for providing support for creating dynamic credentials in [YugabyteDB](https://docs.yugabyte.com).

## Prerequisites

* vault v1.8.0-dev
* yugabyte v2.7.2
* go v1.16.6
* docker v20.10.7

## Installation

The Vault plugin system is documented on the [Vault documentation site](https://www.vaultproject.io/docs/internals/plugins).

You will need to define a plugin directory using the [plugin_directory](https://www.vaultproject.io/docs/configuration#plugin_directory) configuration directive, then place the _vault-plugin-database-yugabytedb_ executable generated above, into the directory.

A sample configuration file for Vault will look like the following:

```HCL
plugin_directory = "plugins"
```

Save this into a file named `vault-server-conf` for later use.

> Please note: Versions v0.2.0 onwards of this plugin are incompatible with Vault versions before 1.6.0 due to an update of the database plugin interface.

Let's run the Vault server and YugabyteDB and start to register our plugin:

```shell
$ vault server -dev -dev-root-token-id root -config vault-server.conf
...
2021-08-02T12:52:55.401+0300 [INFO]  core: upgrading plugin information: plugins=[]
2021-08-02T12:52:55.401+0300 [INFO]  core: successfully setup plugin catalog: plugin-directory=/Users/batuhan.apaydin/plugins
...
```

Run the YugabyteDB in Docker container with the following flags:

```shell
$ docker container run -d --name yugabyte  -p7000:7000 -p9000:9000 -p5433:5433 -p9042:9042 \
 -v ~/yb_data:/home/yugabyte/var \
 yugabytedb/yugabyte:latest bin/yugabyted start \
 --daemon=false --tserver_flags "ysql_enable_auth=true"
```

> You can access all the command flags of yugabyted from this [link](https://docs.yugabyte.com/latest/reference/configuration/yugabyted/).

Sample commands for registering and starting to use the plugin:

```shell
$ git clone https://gitlab.trendyol.com/platform/base/poc/vault-plugin-database-yugabytedb.git 

$ cd vault-plugin-database-yugabytedb

$ go build -o ~/plugins/vault-plugin-database-yugabytedb .

$ export SHA256=$(sha256sum ~/plugins/vault-plugin-database-yugabytedb | cut -d' ' -f1)

$ export VAULT_ADDR="http://localhost:8200"

$ export VAULT_TOKEN="root"

$ vault secrets enable database

$ vault write sys/plugins/catalog/database/vault-plugin-database-yugabytedb \                                         ─╯
    sha256=$SHA256 \
    command="vault-plugin-database-yugabytedb"

$ vault write database/config/yugabytedb plugin_name=vault-plugin-database-yugabytedb \
    host="127.0.0.1" \
    port=5433 \
    username="yugabyte" \
    password="yugabyte" \
    db="yugabyte" \
    allowed_roles="*"

$ vault write database/roles/my-first-role \                                                                          ─╯
    db_name=yugabytedb \
    creation_statements="CREATE ROLE \"{{username}}\" WITH PASSWORD '{{password}}' NOINHERIT LOGIN; \
       GRANT ALL ON DATABASE \"yugabyte\" TO \"{{username}}\";" \
    default_ttl="1h" \
    max_ttl="24h"
```

Once you completed all of the steps above, you should now ready to create your first dynamic credentials:

```shell
$ vault read database/creds/my-first-role
Key                Value
---                -----
lease_id           database/creds/my-first-role/fHFfOJnnpRWh12h4Psv0cbP2
lease_duration     1h
lease_renewable    true
password           -U6NdLMtqR5q9tuWmriI
username           V_TOKEN_MY-FIRST-ROLE_D3VYCRJQM2G74TFSM1DO_1627894589
```
