module go-admin

go 1.18

replace github.com/go-redis/redis/extra/rediscmd/v8 v8.11.5 => gitlab.deeplycurious.cn/anyidong/redis/extra/rediscmd/v8 v8.11.5-u1

require (
	github.com/alibaba/sentinel-golang v1.0.4
	github.com/alibaba/sentinel-golang/pkg/adapters/gin v0.0.0-20220808015021-c5f1f1d055c5
	github.com/aliyun/aliyun-oss-go-sdk v0.0.0-20190307165228-86c17b95fcd5
	github.com/bytedance/go-tagexpr/v2 v2.7.12
	github.com/casbin/casbin/v2 v2.51.2
	github.com/gin-gonic/gin v1.8.1
	github.com/go-admin-team/go-admin-core v1.4.1-0.20220809101213-21187928f7d9
	github.com/go-admin-team/go-admin-core/sdk v1.4.1-0.20220809101213-21187928f7d9
	github.com/go-redis/redis/extra/redisotel/v8 v8.11.5
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-resty/resty/v2 v2.7.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.4.2
	github.com/huaweicloud/huaweicloud-sdk-go-obs v3.21.12+incompatible
	github.com/minio/minio-go/v7 v7.0.45
	github.com/mssola/user_agent v0.5.2
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/qiniu/go-sdk/v7 v7.11.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/shirou/gopsutil/v3 v3.22.1
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/cobra v1.0.0
	github.com/tjfoc/gmsm v1.4.1
	github.com/unrolled/secure v1.0.8
	github.com/uptrace/opentelemetry-go-extra/otelgorm v0.1.15
	github.com/uptrace/opentelemetry-go-extra/otellogrus v0.1.15
	github.com/uptrace/uptrace-go v1.7.1
	github.com/xuri/excelize/v2 v2.6.0
	go.mongodb.org/mongo-driver v1.11.2
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.34.0
	go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo v0.34.0
	go.opentelemetry.io/otel v1.9.0
	go.opentelemetry.io/otel/metric v0.31.0
	go.opentelemetry.io/otel/trace v1.9.0
	golang.org/x/crypto v0.0.0-20220722155217-630584e8d5aa
	gorm.io/driver/mysql v1.3.5
	gorm.io/driver/postgres v1.3.8
	gorm.io/driver/sqlite v1.3.6
	gorm.io/driver/sqlserver v1.3.2
	gorm.io/gorm v1.23.8
)

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible // indirect
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/bsm/redislock v0.5.0 // indirect
	github.com/casbin/redis-watcher/v2 v2.0.0-20220614104201-0e70bf2be930 // indirect
	github.com/cenkalti/backoff/v4 v4.1.3 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/chanxuehong/rand v0.0.0-20201110082127-2f19a1bdd973 // indirect
	github.com/chanxuehong/wechat v0.0.0-20201110083048-0180211b69fd // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/denisenkom/go-mssqldb v0.12.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/fatih/color v1.9.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/git-chglog/git-chglog v0.0.0-20190611050339-63a4e637021f // indirect
	github.com/go-admin-team/go-admin-core/plugins/logger/zap v0.0.0-20210610020726-2db73adb505d // indirect
	github.com/go-admin-team/gorm-adapter/v3 v3.7.8-0.20220809100335-eaf9f67b3d21 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/go-playground/validator/v10 v10.10.0 // indirect
	github.com/go-redis/redis/extra/rediscmd/v8 v8.11.5 // indirect
	github.com/go-redis/redis/v7 v7.4.0 // indirect
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/goccy/go-json v0.9.7 // indirect
	github.com/golang-jwt/jwt/v4 v4.4.2 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.0.0-20170517235910-f1bb20e5a188 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.10.0 // indirect
	github.com/henrylee2cn/ameda v1.4.10 // indirect
	github.com/henrylee2cn/goutil v0.0.0-20210127050712-89660552f6f8 // indirect
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.12.1 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/pgtype v1.11.0 // indirect
	github.com/jackc/pgx/v4 v4.16.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/klauspost/cpuid/v2 v2.1.0 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-sqlite3 v1.14.12 // indirect
	github.com/mattn/goveralls v0.0.2 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/mojocn/base64Captcha v1.3.1 // indirect
	github.com/montanaflynn/stats v0.0.0-20171201202039-1bf9dbcd8cbe // indirect
	github.com/nsqio/go-nsq v1.0.8 // indirect
	github.com/nyaruka/phonenumbers v1.0.55 // indirect
	github.com/pelletier/go-toml/v2 v2.0.1 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.26.0 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.1 // indirect
	github.com/robinjoseph08/redisqueue/v2 v2.1.0 // indirect
	github.com/rs/xid v1.4.0 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/shamsher31/goimgext v1.0.0 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/pflag v1.0.3 // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.3.0 // indirect
	github.com/tsuyoshiwada/go-gitcmd v0.0.0-20180205145712-5f1f5f9475df // indirect
	github.com/ugorji/go/codec v1.2.7 // indirect
	github.com/uptrace/opentelemetry-go-extra/otelsql v0.1.15 // indirect
	github.com/uptrace/opentelemetry-go-extra/otelutil v0.1.15 // indirect
	github.com/urfave/cli v1.22.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.1 // indirect
	github.com/xdg-go/stringprep v1.0.3 // indirect
	github.com/xuri/efp v0.0.0-20220407160117-ad0f7a785be8 // indirect
	github.com/xuri/nfp v0.0.0-20220409054826-5e722a1d9e22 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.opentelemetry.io/contrib/instrumentation/runtime v0.32.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.7.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.30.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.30.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.7.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.7.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.7.0 // indirect
	go.opentelemetry.io/otel/sdk v1.7.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.30.0 // indirect
	go.opentelemetry.io/proto/otlp v0.16.0 // indirect
	go.uber.org/atomic v1.6.0 // indirect
	go.uber.org/multierr v1.5.0 // indirect
	go.uber.org/zap v1.15.0 // indirect
	golang.org/x/image v0.0.0-20211028202545-6944b10bf410 // indirect
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	golang.org/x/tools v0.1.5 // indirect
	google.golang.org/genproto v0.0.0-20220505152158-f39f71e6c8f3 // indirect
	google.golang.org/grpc v1.46.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/AlecAivazis/survey.v1 v1.8.5 // indirect
	gopkg.in/ini.v1 v1.66.6 // indirect
	gopkg.in/kyokomi/emoji.v1 v1.5.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gorm.io/plugin/dbresolver v1.2.2 // indirect
)
