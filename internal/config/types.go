package config

// Root is the top-level configuration structure.
type Root struct {
	Debug    bool           `mapstructure:"debug" yaml:"debug"`
	Wormhole WormholeConfig `mapstructure:"wormhole" yaml:"wormhole"`
}

// WormholeConfig holds wormhole/relay configuration.
type WormholeConfig struct {
	ActiveRelay string            `mapstructure:"active_relay" yaml:"active_relay"`
	Relays      map[string]string `mapstructure:"relays" yaml:"relays"`
}

// GetActiveRelayAddr returns the address of the active relay.
func (w *WormholeConfig) GetActiveRelayAddr() string {
	if w.Relays == nil {
		return ""
	}
	return w.Relays[w.ActiveRelay]
}
