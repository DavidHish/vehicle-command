// NOT FROM TESLA. Together navigation extension.
//
// This entire file is an addition to the upstream teslamotors/vehicle-command
// repository. It contains no cryptography, no session handling, no counter or
// epoch logic and no transport: it builds one protobuf action and hands it to
// Tesla's own executeCarServerAction, which does all of that unchanged. If
// upstream ever publishes a navigation action, delete this file and the
// NavigationGpsRequest block in car_server.proto and nothing else needs to
// move.
//
// See the provenance and warnings on NavigationGpsRequest in
// pkg/protocol/protobuf/car_server.proto before trusting any of this.

package vehicle

import (
	"context"

	carserver "github.com/teslamotors/vehicle-command/pkg/protocol/protobuf/carserver"
)

// TripOrder selects how a coordinate joins the vehicle's current trip.
//
// It is a raw int32 rather than the generated enum on purpose. The numbering is
// disputed: the published definition says replace is 1, prepend 2 and append 3,
// while the same project's own notes say app-side evidence numbers them one
// lower throughout. Sending a bare integer lets both be tested against a real
// car without regenerating protobufs or redeploying a different binary, which
// is the only way this question actually gets settled.
type TripOrder int32

// The published numbering. Names, not gospel — see above.
const (
	TripOrderUnknown TripOrder = 0
	TripOrderReplace TripOrder = 1
	TripOrderPrepend TripOrder = 2
	TripOrderAppend  TripOrder = 3
)

// NavigateToCoordinates sends a single latitude/longitude to the vehicle's
// navigation as a signed command, joining the current trip according to order.
//
// A multi-stop route is built by calling this once per point: the first with a
// replace order to clear whatever the car was doing, and each subsequent point
// with an append order. Whether the vehicle actually accumulates stops that way
// is the open question this extension exists to answer; nobody has published a
// transcript either way.
//
// The coordinates are sent exactly as given. Nothing here rounds them, snaps
// them to a road, or resolves them to an address, which is the whole point: the
// interesting points are on a ramp or in a lane and have no address at all.
func (v *Vehicle) NavigateToCoordinates(ctx context.Context, lat, lon float64, order TripOrder) error {
	return v.executeCarServerAction(ctx,
		&carserver.Action_VehicleAction{
			VehicleAction: &carserver.VehicleAction{
				VehicleActionMsg: &carserver.VehicleAction_NavigationGpsRequest{
					NavigationGpsRequest: &carserver.NavigationGpsRequest{
						Lat:   lat,
						Lon:   lon,
						Order: carserver.NavigationGpsRequest_RemoteNavTripOrder(order),
					},
				},
			},
		})
}
