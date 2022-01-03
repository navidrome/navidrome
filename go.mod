module github.com/navidrome/navidrome

go 1.16

require (
	code.cloudfoundry.org/go-diodes v0.0.0-20190809170250-f77fb823c7ee
	github.com/ClickHouse/clickhouse-go v1.4.5 // indirect
	github.com/Masterminds/squirrel v1.5.1
	github.com/ReneKroon/ttlcache/v2 v2.9.0
	github.com/astaxie/beego v1.12.3
	github.com/bradleyjkemp/cupaloy v2.3.0+incompatible
	github.com/cespare/reflex v0.3.1
	github.com/deluan/rest v0.0.0-20211101235434-380523c4bb47
	github.com/denisenkom/go-mssqldb v0.10.0 // indirect
	github.com/dhowden/tag v0.0.0-20200412032933-5d76b8eaae27
	github.com/disintegration/imaging v1.6.2
	github.com/djherbis/atime v1.1.0
	github.com/djherbis/fscache v0.10.2-0.20201024185917-a0daa9e52747
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/structs v1.1.0
	github.com/go-chi/chi/v5 v5.0.7
	github.com/go-chi/cors v1.2.0
	github.com/go-chi/httprate v0.5.2
	github.com/go-chi/jwtauth/v5 v5.0.2
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/golangci/golangci-lint v1.43.0
	github.com/google/uuid v1.3.0
	github.com/google/wire v0.5.0
	github.com/kennygrant/sanitize v0.0.0-20170120101633-6a0bfdde8629
	github.com/kr/pretty v0.3.0
	github.com/lestrrat-go/jwx v1.2.11
	github.com/matoous/go-nanoid v1.5.0
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/mattn/go-zglob v0.0.3
	github.com/microcosm-cc/bluemonday v1.0.17
	github.com/mileusna/useragent v1.0.2
	github.com/oklog/run v1.1.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.17.0
	github.com/pressly/goose v2.7.0+incompatible
	github.com/robfig/cron/v3 v3.0.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.9.0
	github.com/stretchr/testify v1.7.0
	github.com/unrolled/secure v1.0.9
	github.com/xrash/smetrics v0.0.0-20200730060457-89a2a8a1fb0b
	github.com/ziutek/mymysql v1.5.4 // indirect
	golang.org/x/image v0.0.0-20191009234506-e7c1f5e7dbb8
	golang.org/x/net v0.0.0-20210805182204-aaa1db679c0d
	golang.org/x/tools v0.1.7
	gopkg.in/djherbis/atime.v1 v1.0.0 // indirect
	gopkg.in/djherbis/stream.v1 v1.3.1
)

replace github.com/dhowden/tag => github.com/wader/tag v0.0.0-20200426234345-d072771f6a51
