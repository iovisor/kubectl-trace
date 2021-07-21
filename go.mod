module github.com/iovisor/kubectl-trace

go 1.15

require (
	cloud.google.com/go/storage v1.16.0
	github.com/creack/pty v1.1.13
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/fntlnz/mountinfo v0.0.0-20171106231217-40cb42681fad
	github.com/fsouza/fake-gcs-server v1.29.0
	github.com/kr/pretty v0.2.1 // indirect
	github.com/mholt/archiver/v3 v3.5.0
	github.com/spf13/afero v1.3.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	golang.org/x/mod v0.4.2
	google.golang.org/api v0.50.0
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/cli-runtime v0.19.3
	k8s.io/client-go v0.19.3
	k8s.io/kubectl v0.19.3
	sigs.k8s.io/kind v0.9.0
	sigs.k8s.io/yaml v1.2.0
)
