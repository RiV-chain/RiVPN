package defaults

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/RiV-chain/RiVPN/src/config"
	"github.com/hjson/hjson-go"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/text/encoding/unicode"
)

var defaultConfig = ""

type defaultParameters struct {
	//Network domain
	DefaultTunnelRouting config.TunnelRoutingConfig
}

// Defines which parameters are expected by default for configuration on a
// specific platform. These values are populated in the relevant defaults_*.go
// for the platform being targeted. They must be set.
type platformDefaultParameters struct {
	DefaultConfigFile string
}

// Defines defaults for the all platforms.
func Define() defaultParameters {
	return defaultParameters{

		// TunnelRouting
		DefaultTunnelRouting: config.TunnelRoutingConfig{
			Enable: false,
		},
	}
}

func GetDefaults() platformDefaultParameters {
	defaults := getDefaults()
	if defaultConfig != "" {
		defaults.DefaultConfigFile = defaultConfig
	}
	return defaults
}

// Generates default configuration and returns a pointer to the resulting
// NodeConfig. This is used when outputting the -genconf parameter and also when
// using -autoconf.
func GenerateConfig() *config.NodeConfig {
	// Get the defaults for the platform.
	GetDefaults()
	// Create a node configuration and populate it.
	cfg := new(config.NodeConfig)
	cfg.TunnelRoutingConfig = Define().DefaultTunnelRouting

	return cfg
}

func ReadConfig(useconffile string) (*config.NodeConfig, error) {
	// Use a configuration file. If -useconf, the configuration will be read
	// from stdin. If -useconffile, the configuration will be read from the
	// filesystem.
	var conf []byte
	var err error
	if useconffile != "" {
		// Read the file from the filesystem
		conf, err = os.ReadFile(useconffile)
	} else {
		// Read the file from stdin.
		conf, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		return nil, err
	}
	// If there's a byte order mark - which Windows 10 is now incredibly fond of
	// throwing everywhere when it's converting things into UTF-16 for the hell
	// of it - remove it and decode back down into UTF-8. This is necessary
	// because hjson doesn't know what to do with UTF-16 and will panic
	if bytes.Equal(conf[0:2], []byte{0xFF, 0xFE}) ||
		bytes.Equal(conf[0:2], []byte{0xFE, 0xFF}) {
		utf := unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
		decoder := utf.NewDecoder()
		conf, err = decoder.Bytes(conf)
		if err != nil {
			return nil, err
		}
	}
	// Generate a new configuration - this gives us a set of sane defaults -
	// then parse the configuration we loaded above on top of it. The effect
	// of this is that any configuration item that is missing from the provided
	// configuration will use a sane default.
	cfg := GenerateConfig()
	var dat map[string]interface{}
	if err := hjson.Unmarshal(conf, &dat); err != nil {
		return nil, err
	}
	// Sanitise the config
	confJson, err := json.Marshal(dat)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(confJson, &cfg); err != nil {
		return nil, err
	}
	// Overlay our newly mapped configuration onto the autoconf node config that
	// we generated above.
	if err = mapstructure.Decode(dat, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func WriteConfig(confFn string, cfg *config.NodeConfig) error {
	bs, err := hjson.Marshal(cfg)
	if err != nil {
		return err
	}
	err = os.WriteFile(confFn, bs, 0644)
	if err != nil {
		return err
	}
	return nil
}
