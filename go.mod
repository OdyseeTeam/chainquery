module github.com/lbryio/chainquery

go 1.15

replace github.com/btcsuite/btcd => github.com/lbryio/lbrycrd.go v0.0.0-20200203050410-e1076f12bf19

require (
	github.com/btcsuite/btcd v0.0.0-20190213025234-306aecffea32
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/fatih/color v1.7.0
	github.com/fsnotify/fsnotify v1.4.7
	github.com/go-ini/ini v1.48.0
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gofrs/uuid v3.2.0+incompatible // indirect
	github.com/golang/protobuf v1.4.3
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/schema v1.0.2 // indirect
	github.com/jmoiron/sqlx v0.0.0-20170430194603-d9bd385d68c0
	github.com/johntdyer/slackrus v0.0.0-20180518184837-f7aae3243a07
	github.com/jteeuwen/go-bindata v3.0.7+incompatible
	github.com/lbryio/errors.go v0.0.0-20180223142025-ad03d3cc6a5c
	github.com/lbryio/lbry.go v1.1.2
	github.com/lbryio/lbry.go/v2 v2.7.1
	github.com/lbryio/lbryschema.go v0.0.0-20190602173230-6d2f69a36f46
	github.com/lbryio/ozzo-validation v0.0.0-20170512160344-202201e212ec
	github.com/lbryio/sockety v0.0.0-20210708201924-f21400c148bb
	github.com/lbryio/types v0.0.0-20201019032447-f0b4476ef386
	github.com/lib/pq v1.1.1 // indirect
	github.com/mattn/go-sqlite3 v1.10.0 // indirect
	github.com/mitchellh/mapstructure v1.1.2
	github.com/pkg/errors v0.9.1
	github.com/pkg/profile v1.3.0
	github.com/prometheus/client_golang v1.10.0
	github.com/rubenv/sql-migrate v0.0.0-20170330050058-38004e7a77f2
	github.com/sfreiberg/gotwilio v0.0.0-20180612161623-8fb7259ba8bf
	github.com/shopspring/decimal v0.0.0-20191009025716-f1972eb1d1f5
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cast v1.3.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.7.1
	github.com/volatiletech/inflect v0.0.0-20170731032912-e7201282ae8d // indirect
	github.com/volatiletech/null v8.0.0+incompatible
	github.com/volatiletech/sqlboiler v3.4.0+incompatible
	github.com/ziutek/mymysql v1.5.4 // indirect
	golang.org/x/net v0.0.0-20200625001655-4c5254603344
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	gopkg.in/DATA-DOG/go-sqlmock.v1 v1.3.0 // indirect
	gopkg.in/gorp.v1 v1.7.1 // indirect
)
