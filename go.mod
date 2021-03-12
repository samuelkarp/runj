module go.sbk.wtf/runj

go 1.14

require (
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/Microsoft/hcsshim v0.8.11 // indirect
	github.com/cheggaaa/pb/v3 v3.0.5
	github.com/containerd/containerd v1.4.3
	github.com/containerd/go-runc v0.0.0-20200220073739-7016d3ce2328
	github.com/gogo/protobuf v1.3.1
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.0.0
	github.com/stretchr/testify v1.5.1
	golang.org/x/sys v0.0.0-20201202213521-69691e467435
	google.golang.org/grpc v1.34.0 // indirect
	gotest.tools/v3 v3.0.3 // indirect
)

replace github.com/containerd/containerd => github.com/samuelkarp/containerd v0.2.4-0.20201218175053-b667c15ed877
