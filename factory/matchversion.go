package factory

import (
	"sync"

	"github.com/spf13/pflag"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubernetes/pkg/version"
)

const (
	flagMatchBinaryVersion = "match-server-version"
)

// MatchVersionFlags is for setting the "match server version" function.
type MatchVersionFlags struct {
	Delegate genericclioptions.RESTClientGetter

	RequireMatchedServerVersion bool
	checkServerVersion          sync.Once
	matchesServerVersionErr     error
}

var _ genericclioptions.RESTClientGetter = &MatchVersionFlags{}

func (f *MatchVersionFlags) checkMatchingServerVersion() error {
	f.checkServerVersion.Do(func() {
		if !f.RequireMatchedServerVersion {
			return
		}
		discoveryClient, err := f.Delegate.ToDiscoveryClient()
		if err != nil {
			f.matchesServerVersionErr = err
			return
		}
		f.matchesServerVersionErr = discovery.MatchesServerVersion(version.Get(), discoveryClient)
	})

	return f.matchesServerVersionErr
}

// ToRESTConfig implements RESTClientGetter.
// Returns a REST client configuration based on a provided path
// to a .kubeconfig file, loading rules, and config flag overrides.
// Expects the AddFlags method to have been called.
func (f *MatchVersionFlags) ToRESTConfig() (*rest.Config, error) {
	if err := f.checkMatchingServerVersion(); err != nil {
		return nil, err
	}
	clientConfig, err := f.Delegate.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	// TODO we should not have to do this.  It smacks of something going wrong.
	setKubernetesDefaults(clientConfig)
	return clientConfig, nil
}

func (f *MatchVersionFlags) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return f.Delegate.ToRawKubeConfigLoader()
}

func (f *MatchVersionFlags) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	if err := f.checkMatchingServerVersion(); err != nil {
		return nil, err
	}
	return f.Delegate.ToDiscoveryClient()
}

// RESTMapper returns a mapper.
func (f *MatchVersionFlags) ToRESTMapper() (meta.RESTMapper, error) {
	if err := f.checkMatchingServerVersion(); err != nil {
		return nil, err
	}
	return f.Delegate.ToRESTMapper()
}

func (f *MatchVersionFlags) AddFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&f.RequireMatchedServerVersion, flagMatchBinaryVersion, f.RequireMatchedServerVersion, "Require server version to match client version")
}

func NewMatchVersionFlags(delegate genericclioptions.RESTClientGetter) *MatchVersionFlags {
	return &MatchVersionFlags{
		Delegate: delegate,
	}
}
