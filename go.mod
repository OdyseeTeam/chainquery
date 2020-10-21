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
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/schema v1.0.2 // indirect
	github.com/hashicorp/hcl v0.0.0-20180404174102-ef8a98b0bbce // indirect
	github.com/jmoiron/sqlx v0.0.0-20170430194603-d9bd385d68c0
	github.com/johntdyer/slack-go v0.0.0-20180213144715-95fac1160b22 // indirect
	github.com/johntdyer/slackrus v0.0.0-20170926115001-3992f319fd0a
	github.com/jteeuwen/go-bindata v3.0.7+incompatible
	github.com/lbryio/lbry.go v1.1.2
	github.com/lbryio/lbry.go/v2 v2.4.7-0.20200203053542-c4772e61c565
	github.com/lbryio/lbryschema.go v0.0.0-20190602173230-6d2f69a36f46
	github.com/lbryio/ozzo-validation v0.0.0-20170323141101-d1008ad1fd04
	github.com/lbryio/types v0.0.0-20191009145016-1bb8107e04f8
	github.com/lib/pq v1.1.1 // indirect
	github.com/magiconair/properties v1.8.0 // indirect
	github.com/mattn/go-sqlite3 v1.10.0 // indirect
	github.com/mitchellh/mapstructure v1.1.2
	github.com/pelletier/go-toml v1.2.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/pkg/profile v1.3.0
	github.com/prometheus/client_golang v1.8.0
	github.com/rubenv/sql-migrate v0.0.0-20170330050058-38004e7a77f2
	github.com/sfreiberg/gotwilio v0.0.0-20180612161623-8fb7259ba8bf
	github.com/shopspring/decimal v0.0.0-20191009025716-f1972eb1d1f5
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/afero v1.1.1 // indirect
	github.com/spf13/cast v1.3.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/jwalterweatherman v0.0.0-20180109140146-7c0cea34c8ec // indirect
	github.com/spf13/pflag v1.0.1
	github.com/spf13/viper v1.0.2
	github.com/volatiletech/inflect v0.0.0-20170731032912-e7201282ae8d // indirect
	github.com/volatiletech/null v8.0.0+incompatible
	github.com/volatiletech/sqlboiler v3.4.0+incompatible
	github.com/ziutek/mymysql v1.5.4 // indirect
	golang.org/x/net v0.0.0-20200625001655-4c5254603344
	golang.org/x/oauth2 v0.0.0-20190226205417-e64efc72b421
	gopkg.in/DATA-DOG/go-sqlmock.v1 v1.3.0 // indirect
	gopkg.in/gorp.v1 v1.7.1 // indirect
)
