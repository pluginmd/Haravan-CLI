// Package cmdutil provides the Factory DI container passed to every command.
package cmdutil

import (
	"github.com/pluginmd/haravan-cli/internal/config"
	"github.com/pluginmd/haravan-cli/internal/iostreams"
)

// Factory is a lightweight DI container injected into each Cobra command.
// Providers are lazy so commands don't pay for resources they don't use.
type Factory struct {
	IOStreams *iostreams.IOStreams
	Config    func() (*config.Config, error)
}

func New() *Factory {
	return &Factory{
		IOStreams: iostreams.System(),
		Config:    config.Load,
	}
}
