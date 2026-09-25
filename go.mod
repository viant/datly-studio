module github.com/viant/datly-studio

replace github.com/viant/datly => ../datly

replace github.com/viant/xdatly => ../xdatly

replace github.com/viant/sqlx => ../sqlx

go 1.25.8

require modernc.org/sqlite v1.45.0

require (
	github.com/go-sql-driver/mysql v1.7.0
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/lib/pq v1.10.6
	github.com/mattn/go-sqlite3 v1.14.16
	github.com/viant/bigquery v0.5.3
	github.com/viant/bindly v0.2.1-0.20260915164201-7cf35da5bd9a
	github.com/viant/datly v1.0.1-0.20260924050758-f968896067ed
	github.com/viant/jsonrpc v0.25.0
	github.com/viant/mcp v0.24.0
	github.com/viant/mcp-protocol v0.19.0
	github.com/viant/scy v0.35.1-0.20260914041206-699e6c909726
	github.com/viant/sqlx v0.26.1-0.20260925010754-62fc9f049795
	github.com/viant/x v0.5.1-0.20260915043005-1bc42b9eef47
	github.com/viant/xdatly v1.0.1-0.20260925011738-7249dcd4bf03
	go.yaml.in/yaml/v3 v3.0.5
	golang.org/x/mod v0.37.0
	golang.org/x/oauth2 v0.36.0
)

require (
	cloud.google.com/go/auth v0.20.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/aerospike/aerospike-client-go v4.5.2+incompatible // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/francoispqt/gojay v1.2.13 // indirect
	github.com/go-errors/errors v1.5.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/google/pprof v0.0.0-20260115054156-294ebfa9ad83 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.16 // indirect
	github.com/googleapis/gax-go/v2 v2.22.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/lestrrat-go/backoff/v2 v2.0.8 // indirect
	github.com/lestrrat-go/blackmagic v1.0.2 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/jwx v1.2.29 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mazznoer/csscolorparser v0.1.3 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.3 // indirect
	github.com/viant/afs v1.30.1-0.20260914155934-90e95d483861 // indirect
	github.com/viant/afsc v1.18.0 // indirect
	github.com/viant/gmetric v0.3.2 // indirect
	github.com/viant/gosh v0.2.1 // indirect
	github.com/viant/govalidator v0.3.4-0.20260913213120-027a9dd54d73 // indirect
	github.com/viant/igo v0.2.0 // indirect
	github.com/viant/parsly v0.3.3 // indirect
	github.com/viant/sqlparser v0.13.1-0.20260921232033-929f6f50ccce // indirect
	github.com/viant/structology v0.10.1-0.20260915165514-89d421971772 // indirect
	github.com/viant/structql v0.5.4 // indirect
	github.com/viant/tagly v0.3.1-0.20260914020630-eadee36c3630 // indirect
	github.com/viant/toolbox v0.39.0 // indirect
	github.com/viant/velty v0.4.1-0.20260915052314-fca47c0595f8 // indirect
	github.com/viant/xlsy v0.3.1 // indirect
	github.com/viant/xmlify v0.1.2-0.20260914155716-e525a8788fd0 // indirect
	github.com/viant/xreflect v0.7.5-0.20260314170600-13f09f37d46e // indirect
	github.com/viant/xunsafe v0.11.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/xuri/efp v0.0.0-20230802181842-ad255f2331ca // indirect
	github.com/xuri/excelize/v2 v2.8.0 // indirect
	github.com/xuri/nfp v0.0.0-20230819163627-dc951e3ffe1a // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/exp v0.0.0-20251023183803-a4bb9ffd2546 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/tools v0.47.0 // indirect
	google.golang.org/api v0.283.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/grpc v1.81.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.67.6 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)
