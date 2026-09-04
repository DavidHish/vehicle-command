// NOT FROM TESLA. Together navigation extension.
//
// See the provenance and warnings on NavigationGpsDestinationRequest in
// pkg/protocol/protobuf/car_server.proto.
//
// No cryptography, no sessions, no transport: one protobuf action handed to
// Tesla's own executeCarServerAction.

package vehicle

import (
	"context"

	carserver "github.com/teslamotors/vehicle-command/pkg/protocol/protobuf/carserver"
)

// NavigateToNamedCoordinates sends an exact coordinate together with a name.
//
// This is the only message in Tesla's schema that carries both. Everything else
// that produces a friendly name on the touchscreen does so by way of a Google
// place id, and a place id has its own position — which is how a pin
// deliberately placed on a school's drop-off lane ends up routing to the front
// of the building instead. Position is the whole point of this feature, so a
// message with lat, lon and a name is worth trying.
//
// The coordinates are sent exactly as given: nothing here rounds them, snaps
// them to a road, or resolves them to an address.
//
// Whether `destination` actually DISPLAYS is unknown. It has been sent to a
// real car and acknowledged, and nobody has ever written down what appeared on
// the screen. It is equally consistent with being the label and with being a
// geocoding hint the car consumes and discards, and only the touchscreen
// settles that.
func (v *Vehicle) NavigateToNamedCoordinates(
	ctx context.Context,
	lat, lon float64,
	name string,
	order TripOrder,
) error {
	return v.executeCarServerAction(ctx,
		&carserver.Action_VehicleAction{
			VehicleAction: &carserver.VehicleAction{
				VehicleActionMsg: &carserver.VehicleAction_NavigationGpsDestinationRequest{
					NavigationGpsDestinationRequest: &carserver.NavigationGpsDestinationRequest{
						Lat:         lat,
						Lon:         lon,
						Destination: name,
						Order:       carserver.NavigationGpsDestinationRequest_RemoteNavTripOrder(order),
					},
				},
			},
		})
}
