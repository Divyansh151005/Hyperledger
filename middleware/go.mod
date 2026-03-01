module github.com/hyperledger/medical-middleware

go 1.24.0

require (
	github.com/aws/aws-sdk-go v1.47.0
	github.com/gin-gonic/gin v1.9.1
	github.com/google/uuid v1.6.0
	github.com/hyperledger/fabric-gateway v1.3.0
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/sirupsen/logrus v1.9.3
	golang.org/x/crypto v0.44.0
	google.golang.org/grpc v1.76.0
)

require (
	github.com/bytedance/sonic v1.9.1 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3abad311 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.14.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/hyperledger/fabric-protos-go-apiv2 v0.3.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/miekg/pkcs11 v1.1.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	golang.org/x/arch v0.3.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Collapse all genproto submodule paths to the main module to avoid ambiguity
// This ensures all submodules (like googleapis/rpc) resolve to the same main module version
// This prevents the "used for two different module paths" error
// Note: The submodule is replaced to the main module, allowing Go to resolve packages from the main module
replace google.golang.org/genproto/googleapis/rpc => google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1
