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
	"fmt"

	carserver "github.com/teslamotors/vehicle-command/pkg/protocol/protobuf/carserver"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// TripOrder selects how a coordinate joins the vehicle's current trip.
//
// A raw int32 rather than the generated enum on purpose: the numbering is
// disputed, and sending a bare integer lets every candidate be tried against a
// real car without regenerating protobufs.
type TripOrder int32

// The published numbering. Names, not gospel.
const (
	TripOrderUnknown TripOrder = 0
	TripOrderReplace TripOrder = 1
	TripOrderPrepend TripOrder = 2
	TripOrderAppend  TripOrder = 3
)

// DefaultOrderField is where the published third-party definition puts `order`.
const DefaultOrderField = 3

// NavigateToCoordinates sends a single latitude/longitude to the vehicle's
// navigation as a signed command, using the published field number for order.
func (v *Vehicle) NavigateToCoordinates(ctx context.Context, lat, lon float64, order TripOrder) error {
	return v.NavigateToCoordinatesAt(ctx, lat, lon, order, DefaultOrderField)
}

// NavigateToCoordinatesAt is NavigateToCoordinates with the order carried at a
// chosen protobuf field number.
//
// This exists because of a real observation. Sent against a Model 3 on firmware
// 2026.26.6, this command reliably delivered its COORDINATES: each point
// appeared on the touchscreen as a destination offer, at exactly the right
// place. But the order value had no effect whatsoever — replace, prepend and
// append all behaved identically, at every candidate numbering, whether or not
// navigation was already running. Every point simply replaced the previous
// offer and only the last survived.
//
// Coordinates arriving correctly while order does nothing is the signature of
// order sitting at the WRONG FIELD NUMBER: protobuf discards an unknown field
// silently, so a misplaced value is indistinguishable from one that was never
// sent. lat and lon are evidently at 1 and 2, since they arrive intact. Only
// order's position is in doubt, and the published definition guessed it.
//
// So rather than rebuild for each guess, the field number is a parameter. The
// value is written directly into the message's unknown-fields region, which is
// how protobuf carries a field the schema does not describe: the wire format is
// identical to a field that was declared there all along.
//
// When the right number is found, put it in the .proto, delete this, and use
// the generated field like a normal person.
func (v *Vehicle) NavigateToCoordinatesAt(
	ctx context.Context,
	lat, lon float64,
	order TripOrder,
	orderField int,
) error {
	if orderField < 1 || orderField > 536870911 {
		return fmt.Errorf("order field number %d is not a valid protobuf field", orderField)
	}

	req := &carserver.NavigationGpsRequest{Lat: lat, Lon: lon}

	if orderField == DefaultOrderField {
		// The declared field: set it normally so the generated code owns it.
		req.Order = carserver.NavigationGpsRequest_RemoteNavTripOrder(order)
	} else {
		// Anywhere else: append a varint field by hand. protowire produces
		// exactly the bytes protoc-gen-go would have, so the vehicle cannot
		// tell this from a declared field.
		raw := protowire.AppendTag(nil, protowire.Number(orderField), protowire.VarintType)
		raw = protowire.AppendVarint(raw, uint64(order))
		req.ProtoReflect().SetUnknown(protoreflect.RawFields(raw))
	}

	return v.executeCarServerAction(ctx,
		&carserver.Action_VehicleAction{
			VehicleAction: &carserver.VehicleAction{
				VehicleActionMsg: &carserver.VehicleAction_NavigationGpsRequest{
					NavigationGpsRequest: req,
				},
			},
		})
}
