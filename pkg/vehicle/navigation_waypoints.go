// NOT FROM TESLA. Together navigation extension.
//
// The multi-stop counterpart to navigation_gps.go. See the provenance notes on
// NavigationWaypointsRequest in pkg/protocol/protobuf/car_server.proto.
//
// No cryptography, no sessions, no transport: one protobuf action handed to
// Tesla's own executeCarServerAction.

package vehicle

import (
	"context"
	"errors"

	carserver "github.com/teslamotors/vehicle-command/pkg/protocol/protobuf/carserver"
)

// NavigateToWaypoints sends an entire ordered trip in ONE signed command.
//
// This is the difference that matters. NavigateToCoordinates carries a single
// destination, so sending a route point by point just overwrites one slot five
// times — which is exactly what a real car did: five destinations in sequence,
// each replacing the last, and an order field that never appeared to do
// anything because there was no list for it to order.
//
// `waypoints` is passed to the vehicle verbatim. The one published format is a
// comma-separated list of `refId:<Google Place ID>`, the last entry being the
// final destination and the rest intermediate stops. Whether coordinates work
// too is untested by anyone whose evidence we could trace, so the caller
// chooses the encoding and we do not second-guess it here.
//
// startSoePct and arrivalSoePct are Tesla's trip-planner state-of-energy hints,
// in whole percent. Zero leaves them unset, which is what you want unless you
// have a reason.
func (v *Vehicle) NavigateToWaypoints(
	ctx context.Context,
	waypoints string,
	startSoePct, arrivalSoePct int32,
) error {
	if waypoints == "" {
		return errors.New("waypoints string is empty")
	}

	req := &carserver.NavigationWaypointsRequest{Waypoints: waypoints}
	if startSoePct != 0 || arrivalSoePct != 0 {
		req.TripPlanOptions = &carserver.NavigationWaypointsRequest_TripPlanOptions{
			DestinationStartSoe:   startSoePct,
			DestinationArrivalSoe: arrivalSoePct,
		}
	}

	return v.executeCarServerAction(ctx,
		&carserver.Action_VehicleAction{
			VehicleAction: &carserver.VehicleAction{
				VehicleActionMsg: &carserver.VehicleAction_NavigationWaypointsRequest{
					NavigationWaypointsRequest: req,
				},
			},
		})
}
