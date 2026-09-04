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
	"fmt"

	carserver "github.com/teslamotors/vehicle-command/pkg/protocol/protobuf/carserver"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"
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
	return v.NavigateToNamedCoordinatesAt(ctx, lat, lon, name, order, NamedOrderField)
}

// NamedOrderField is where the published layout puts this message's order.
//
// Note it is 4, not 3. NavigationGpsRequest carries lat, lon, order — order at
// 3. This message carries lat, lon, destination, order — order at 4, pushed
// along by the name. A field sweep done against the other message therefore
// says nothing whatsoever about this one, which is easy to miss and was.
const NamedOrderField = 4

// NavigateToNamedCoordinatesAt is NavigateToNamedCoordinates with the order
// carried at a chosen protobuf field number.
//
// This exists because of a confirmed observation and an unanswered question.
//
// CONFIRMED, on a real Model 3 on 2026.26.6: this message DISPLAYS its name.
// Sent with a custom label, the touchscreen showed that label rather than an
// address or a plus code. As far as the public record goes that had never been
// established by anyone — every previous test of this message read the HTTP
// acknowledgement and nobody looked at the screen.
//
// UNANSWERED: whether its order does anything. Sent one point at a time, each
// replaced the last, exactly as the single-destination message does. That is
// either an inert order field or an order value at a field the car does not
// read — indistinguishable from outside, since protobuf discards an unknown
// field in silence.
//
// The prize for answering it is the whole feature: exact coordinates AND the
// names somebody chose AND a multi-stop route, which no other message in the
// schema can offer together. So the field number is a parameter, written into
// the message's unknown-fields region exactly as protoc-gen-go would emit a
// declared one.
func (v *Vehicle) NavigateToNamedCoordinatesAt(
	ctx context.Context,
	lat, lon float64,
	name string,
	order TripOrder,
	orderField int,
) error {
	if orderField < 1 || orderField > 536870911 {
		return fmt.Errorf("order field number %d is not a valid protobuf field", orderField)
	}

	req := &carserver.NavigationGpsDestinationRequest{
		Lat:         lat,
		Lon:         lon,
		Destination: name,
	}

	if orderField == NamedOrderField {
		req.Order = carserver.NavigationGpsDestinationRequest_RemoteNavTripOrder(order)
	} else {
		raw := protowire.AppendTag(nil, protowire.Number(orderField), protowire.VarintType)
		raw = protowire.AppendVarint(raw, uint64(order))
		req.ProtoReflect().SetUnknown(protoreflect.RawFields(raw))
	}

	return v.executeCarServerAction(ctx,
		&carserver.Action_VehicleAction{
			VehicleAction: &carserver.VehicleAction{
				VehicleActionMsg: &carserver.VehicleAction_NavigationGpsDestinationRequest{
					NavigationGpsDestinationRequest: req,
				},
			},
		})
}
