// NOT FROM TESLA. Tests for the Together multi-stop navigation extension.

package vehicle

import (
	"bytes"
	"testing"

	carserver "github.com/teslamotors/vehicle-command/pkg/protocol/protobuf/carserver"
	"google.golang.org/protobuf/proto"
)

func waypointAction(list string) *carserver.Action {
	return &carserver.Action{ActionMsg: &carserver.Action_VehicleAction{
		VehicleAction: &carserver.VehicleAction{
			VehicleActionMsg: &carserver.VehicleAction_NavigationWaypointsRequest{
				NavigationWaypointsRequest: &carserver.NavigationWaypointsRequest{Waypoints: list},
			},
		},
	}}
}

// Tag 90 is the whole bet. A sibling message from the same third-party source
// shipped with the wrong tag and the vehicle silently ignored it while still
// reporting success, so nothing but a wire-level assertion will notice if this
// moves.
func TestWaypointsUsesFieldNinety(t *testing.T) {
	encoded, err := proto.Marshal(waypointAction("refId:ChIJa,refId:ChIJb"))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// 90<<3|2 = 722, which encodes as the varint 0xD2 0x05.
	if !bytes.Contains(encoded, []byte{0xD2, 0x05}) {
		t.Fatalf("encoded action does not carry field 90; bytes = %x", encoded)
	}
}

// The waypoints string reaches the car byte for byte. The caller chooses the
// encoding — place ids, coordinates, whatever we are testing that day — and
// this layer must not normalise, trim or reorder any of it.
func TestWaypointsStringIsVerbatim(t *testing.T) {
	for _, list := range []string{
		"refId:ChIJN1t_tDeuEmsRUsoyG83frY4",
		"refId:ChIJa,refId:ChIJb,refId:ChIJc",
		"38.907235,-77.036912",
		"gps:38.907235,-77.036912,gps:38.812725,-77.193147",
	} {
		encoded, err := proto.Marshal(waypointAction(list))
		if err != nil {
			t.Fatalf("%q: marshal: %v", list, err)
		}
		var decoded carserver.Action
		if err := proto.Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("%q: unmarshal: %v", list, err)
		}
		got := decoded.GetVehicleAction().GetNavigationWaypointsRequest().GetWaypoints()
		if got != list {
			t.Fatalf("waypoints changed in transit: got %q want %q", got, list)
		}
	}
}

// Trip-plan options are optional and must stay absent when unset, rather than
// being sent as zeroes that the planner might read as real targets.
func TestTripPlanOptionsAreOmittedWhenUnset(t *testing.T) {
	req := &carserver.NavigationWaypointsRequest{Waypoints: "refId:ChIJa"}
	if req.GetTripPlanOptions() != nil {
		t.Fatal("trip plan options should be nil when never set")
	}
	req.TripPlanOptions = &carserver.NavigationWaypointsRequest_TripPlanOptions{
		DestinationArrivalSoe: 20,
	}
	if req.GetTripPlanOptions().GetDestinationArrivalSoe() != 20 {
		t.Fatal("arrival state of energy did not round trip")
	}
}

// Adding tag 90 must not have disturbed tag 53, which is a separate command we
// still send.
func TestGpsRequestSurvivesTheWaypointsAddition(t *testing.T) {
	encoded, err := proto.Marshal(action(1, 2, TripOrderReplace))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !bytes.Contains(encoded, []byte{0xAA, 0x03}) {
		t.Fatalf("field 53 moved; bytes = %x", encoded)
	}
}

// Tag 106 is the only message pairing an exact coordinate with a name, so if it
// ever moves, the one path that could give us both silently stops working.
func TestNamedDestinationUsesFieldOneHundredSix(t *testing.T) {
	req := &carserver.Action{ActionMsg: &carserver.Action_VehicleAction{
		VehicleAction: &carserver.VehicleAction{
			VehicleActionMsg: &carserver.VehicleAction_NavigationGpsDestinationRequest{
				NavigationGpsDestinationRequest: &carserver.NavigationGpsDestinationRequest{
					Lat: 38.907235, Lon: -77.036912, Destination: "Oakwood School",
				},
			},
		},
	}}
	encoded, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// 106<<3|2 = 850, which encodes as the varint 0xD2 0x06.
	if !bytes.Contains(encoded, []byte{0xD2, 0x06}) {
		t.Fatalf("action does not carry field 106; bytes = %x", encoded)
	}
	// The name must reach the wire verbatim; that is the entire hypothesis.
	if !bytes.Contains(encoded, []byte("Oakwood School")) {
		t.Fatalf("the destination name did not reach the wire; bytes = %x", encoded)
	}
}

// The coordinates must survive at the precision the editor stores, or the
// message solves the naming problem by reintroducing the position problem.
func TestNamedDestinationKeepsExactCoordinates(t *testing.T) {
	const lat, lon = 38.907235, -77.036912
	req := &carserver.NavigationGpsDestinationRequest{Lat: lat, Lon: lon, Destination: "x"}
	encoded, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back carserver.NavigationGpsDestinationRequest
	if err := proto.Unmarshal(encoded, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Lat != lat || back.Lon != lon {
		t.Fatalf("coordinates changed: got %v,%v", back.Lat, back.Lon)
	}
	if back.Destination != "x" {
		t.Fatalf("name changed: got %q", back.Destination)
	}
}
