# RiVPN

This is an RiV-mesh v0.4 build that re-adds tunnel routing/crypto-key routing (CKR) support. Add a section to your `mesh.conf` like this:

```
FeaturesConfig:
{
  TunnelRouting: {
    Enable: true
    IPv4RemoteSubnets: {
      "a.a.a.a/a": remotepublickey
    }
    IPv6RemoteSubnets: {
      "b::b/b": remotepublickey
    }
  }
}
```

Then use Go 1.18 to build and run:
```
go build -o mesh ./cmd/mesh
./mesh -useconffile ...
```

... or generate an iOS framework with:

```
gomobile bind -target ios -tags mobile -o Mesh.framework \
  github.com/RiV-chain/RiVPN/src/mobile \
  github.com/RiV-chain/RiVPN/src/config
```

... or generate an Android AAR bundle with:

```
gomobile bind -target android -tags mobile -o mesh.aar \
  github.com/RiV-chain/RiVPN/src/mobile \
  github.com/RiV-chain/RiVPN/src/config
```

The main change from the old tunnel routing/CKR support in v0.3 is that you don't need to specify source subnets. Filtering will automatically be applied based on your remote subnets, therefore you'll need to specify the correct remote subnets on both sides.
