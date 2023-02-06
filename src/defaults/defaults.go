package defaults

import (
	"os"

	"github.com/RiV-chain/RiVPN/src/config"
	"github.com/hjson/hjson-go"
)

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
