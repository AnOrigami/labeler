package service

import (
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
)

var (
	seatMeter                  = global.MeterProvider().Meter("scrm_seat_meter")
	seatWSUpDownCounter        syncint64.UpDownCounter
	seatCheckInUpDownCounter   syncint64.UpDownCounter
	seatReadinessUpDownCounter syncint64.UpDownCounter
)

func init() {
	a, err := seatMeter.SyncInt64().UpDownCounter(
		"seat.ws.up_down_counter",
		instrument.WithUnit("1"),
		instrument.WithDescription("seat ws count"),
	)
	if err != nil {
		panic(err)
	}
	seatWSUpDownCounter = a

	b, err := seatMeter.SyncInt64().UpDownCounter(
		"seat.check_state.up_down_counter",
		instrument.WithUnit("1"),
		instrument.WithDescription("seat checkin count"),
	)
	if err != nil {
		panic(err)
	}
	seatCheckInUpDownCounter = b

	c, err := seatMeter.SyncInt64().UpDownCounter(
		"seat.readiness_state.up_down_counter",
		instrument.WithUnit("1"),
		instrument.WithDescription("seat readiness count"),
	)
	if err != nil {
		panic(err)
	}
	seatReadinessUpDownCounter = c
}
