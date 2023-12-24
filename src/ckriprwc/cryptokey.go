package ckriprwc

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gologme/log"

	"github.com/RiV-chain/RiV-mesh/src/core"
	"github.com/RiV-chain/RiVPN/src/config"
)

type cryptokey struct {
	core    *core.Core
	log     *log.Logger
	config  *config.TunnelRoutingConfig
	enabled atomic.Value // bool
	sync.RWMutex
	v4Routes []*route
	v6Routes []*route
}

type route struct {
	Prefix      netip.Prefix
	destination ed25519.PublicKey
}

// Configure the CKR routes. This should only ever be ran by the TUN/TAP actor.
func (c *cryptokey) configure() error {
	// Set enabled/disabled state
	c.setEnabled(c.config.Enable)
	if !c.config.Enable {
		return nil
	}

	c.Lock()
	c.v4Routes = make([]*route, 0, len(c.config.IPv4RemoteSubnets))
	c.v6Routes = make([]*route, 0, len(c.config.IPv6RemoteSubnets))
	c.Unlock()
	i := 0
	for {
		if len(c.core.GetPeers()) > 0 {
			for ipv6, pubkey := range c.config.IPv6RemoteSubnets {
				if err := c.addRemoteSubnet(ipv6, pubkey); err != nil {
					return fmt.Errorf("Error adding routed IPv6 subnet: %w", err)
				}
			}
			for ipv4, pubkey := range c.config.IPv4RemoteSubnets {
				if err := c.addRemoteSubnet(ipv4, pubkey); err != nil {
					return fmt.Errorf("Error adding routed IPv4 subnet: %w", err)
				}
			}
			break
		} else {
			i++
			if i > 10 {
				return fmt.Errorf("No peers has been added")
			}
			time.Sleep(6 * time.Second)
		}
	}
	return nil
}

// Enable or disable crypto-key routing.
func (c *cryptokey) setEnabled(enabled bool) {
	c.enabled.Store(enabled)
}

// Check if crypto-key routing is enabled.
func (c *cryptokey) isEnabled() bool {
	enabled, ok := c.enabled.Load().(bool)
	return ok && enabled
}

// Adds a destination route for the given CIDR to be tunnelled to the node
// with the given BoxPubKey.
func (c *cryptokey) addRemoteSubnet(cidr string, dest string) error {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return err
	}
	if c.isMeshDestination(prefix.Addr()) {
		return errors.New("can't specify RiV-mesh destination as routed subnet")
	}

	c.Lock()
	defer c.Unlock()

	switch {
	case prefix.Addr().Is6():
		for _, route := range c.v6Routes {
			if route.Prefix == prefix {
				return fmt.Errorf("remote subnet already exists for %s", cidr)
			}
		}

	case prefix.Addr().Is4():
		for _, route := range c.v4Routes {
			if route.Prefix == prefix {
				return fmt.Errorf("remote subnet already exists for %s", cidr)
			}
		}

	default:
		return fmt.Errorf("unexpected prefix size")
	}

	if bpk, err := hex.DecodeString(dest); err != nil {
		return fmt.Errorf("hex.DecodeString: %w", err)
	} else {
		destination := make(ed25519.PublicKey, ed25519.PublicKeySize)
		if copy(destination[:], bpk) != ed25519.PublicKeySize {
			return fmt.Errorf("incorrect key length for %q", dest)
		}

		switch {
		case prefix.Addr().Is6():
			c.v6Routes = append(c.v6Routes, &route{
				Prefix:      prefix,
				destination: destination,
			})
			sort.Slice(c.v6Routes, func(i, j int) bool {
				return c.v6Routes[i].Prefix.Bits() > c.v6Routes[j].Prefix.Bits()
			})
			c.log.Infoln("Added routed IPv6 subnet", cidr)

		case prefix.Addr().Is4():
			c.v4Routes = append(c.v4Routes, &route{
				Prefix:      prefix,
				destination: destination,
			})
			sort.Slice(c.v4Routes, func(i, j int) bool {
				return c.v4Routes[i].Prefix.Bits() > c.v4Routes[j].Prefix.Bits()
			})
			c.log.Infoln("Added routed IPv4 subnet", cidr)
		}

		return nil
	}
}

// Looks up the most specific route for the given address (with the address
// length specified in bytes) from the crypto-key routing table. An error is
// returned if the address is not suitable or no route was found.
func (c *cryptokey) getPublicKeyForAddress(addr netip.Addr) (ed25519.PublicKey, error) {
	if !c.isEnabled() {
		return nil, fmt.Errorf("CKR not enabled")
	}
	if c.isMeshDestination(addr) {
		return nil, fmt.Errorf("can't get public key for RiV-mesh route")
	}

	c.RLock()
	defer c.RUnlock()

	switch {
	case addr.Is6():
		for _, route := range c.v6Routes {
			if route.Prefix.Contains(addr) {
				return route.destination, nil
			}
		}

	case addr.Is4():
		for _, route := range c.v4Routes {
			if route.Prefix.Contains(addr) {
				return route.destination, nil
			}
		}

	default:
		return nil, fmt.Errorf("unexpected prefix size")
	}

	return nil, fmt.Errorf("no route to %s", addr.String())
}

func (c *cryptokey) isMeshDestination(ip netip.Addr) bool {
	var addr core.Address
	var snet core.Subnet
	copy(addr[:], ip.AsSlice())
	copy(snet[:], ip.AsSlice())
	return c.core.IsValidAddress(addr) || c.core.IsValidSubnet(snet)
}
