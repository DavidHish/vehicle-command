// NOT FROM TESLA. Tests for the Together navigation extension.

package vehicle

import (
	"bytes"
	"testing"

	carserver "github.com/teslamotors/vehicle-command/pkg/protocol/protobuf/carserver"
	"google.golang.org/protobuf/proto"
)

func action(lat, lon float64, order TripOrder) *carserver.Action {
	return &carserver.Action{ActionMsg: &carserver.Action_VehicleAction{
		VehicleAction: &carserver.VehicleAction{
			VehicleActionMsg: &carserver.VehicleAction_NavigationGpsRequest{
				NavigationGpsRequest: &carserver.NavigationGpsRequest{
					Lat:   lat,
					Lon:   lon,
					Order: carserver.NavigationGpsRequest_RemoteNavTripOrder(order),
				},
			},
		},
	}}
}

// The tag is the single most load-bearing and least verifiable fact in this
// extension. A sibling message in the same third-party source shipped with the
// wrong tag and the vehicle silently ignored it while still reporting success,
// so if the tag ever moves, this test is the only thing that will say so.
func TestNavigationGpsRequestUsesFieldFiftyThree(t *testing.T) {
	encoded, err := proto.Marshal(action(38.9072, -77.0369, TripOrderReplace))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Protobuf key for field 53, wire type 2 (length-delimited): 53<<3|2 = 426,
	// which encodes as the varint 0xAA 0x03.
	key := []byte{0xAA, 0x03}
	if !bytes.Contains(encoded, key) {
		t.Fatalf("encoded action does not carry field 53; bytes = %x", encoded)
	}
}

func TestNavigationGpsRequestRoundTrips(t *testing.T) {
	// Six decimals is the precision the route editor stores, and the precision
	// the whole feature exists to preserve. A lossy encode would round a
	// shaping point off its ramp.
	const lat, lon = 38.907235, -77.036912

	encoded, err := proto.Marshal(action(lat, lon, TripOrderAppend))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded carserver.Action
	if err := proto.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	msg, ok := decoded.GetVehicleAction().VehicleActionMsg.(*carserver.VehicleAction_NavigationGpsRequest)
	if !ok {
		t.Fatalf("decoded into %T, not a navigation request", decoded.GetVehicleAction().VehicleActionMsg)
	}
	got := msg.NavigationGpsRequest
	if got.Lat != lat || got.Lon != lon {
		t.Fatalf("coordinates changed in transit: got %v,%v want %v,%v", got.Lat, got.Lon, lat, lon)
	}
	if got.Order != carserver.NavigationGpsRequest_REMOTE_NAV_TRIP_ORDER_APPEND {
		t.Fatalf("order changed in transit: got %v", got.Order)
	}
}

// The numbering of RemoteNavTripOrder is disputed by the people who published
// it. The bridge must therefore be able to put any integer on the wire so both
// candidate numberings can be tried against a real car. Proto3 preserves
// unknown enum values, and this pins that behaviour so a future regeneration
// cannot quietly clamp it.
func TestTripOrderAcceptsAnyIntegerOnTheWire(t *testing.T) {
	for _, order := range []TripOrder{0, 1, 2, 3, 7} {
		encoded, err := proto.Marshal(action(1, 2, order))
		if err != nil {
			t.Fatalf("marshal order %d: %v", order, err)
		}
		var decoded carserver.Action
		if err := proto.Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("unmarshal order %d: %v", order, err)
		}
		msg := decoded.GetVehicleAction().VehicleActionMsg.(*carserver.VehicleAction_NavigationGpsRequest)
		if int32(msg.NavigationGpsRequest.Order) != int32(order) {
			t.Fatalf("order %d did not survive: got %d", order, msg.NavigationGpsRequest.Order)
		}
	}
}

// The extension must not disturb anything upstream. If adding a message to the
// oneof had shifted an existing tag, this would catch it.
func TestExistingActionTagsAreUndisturbed(t *testing.T) {
	upstream := &carserver.Action{ActionMsg: &carserver.Action_VehicleAction{
		VehicleAction: &carserver.VehicleAction{
			VehicleActionMsg: &carserver.VehicleAction_SetVehicleNameAction{
				SetVehicleNameAction: &carserver.SetVehicleNameAction{VehicleName: "Nikola"},
			},
		},
	}}
	encoded, err := proto.Marshal(upstream)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// setVehicleNameAction is field 54: 54<<3|2 = 434 -> varint 0xB2 0x03.
	if !bytes.Contains(encoded, []byte{0xB2, 0x03}) {
		t.Fatalf("upstream action 54 moved; bytes = %x", encoded)
	}
}
