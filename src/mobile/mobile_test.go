package mobile

import "testing"

func TestStartYggdrasil(t *testing.T) {
	riv := &Mesh{}
	if err := riv.StartAutoconfigure(); err != nil {
		t.Fatalf("Failed to start RiV-mesh: %s", err)
	}
	t.Log("Address:", riv.GetAddressString())
	t.Log("Subnet:", riv.GetSubnetString())
	t.Log("Coords:", riv.GetCoordsString())
	if err := riv.Stop(); err != nil {
		t.Fatalf("Failed to stop RiV-mesh: %s", err)
	}
}
