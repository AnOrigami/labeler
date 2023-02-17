package service

import (
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
)

var (
	ctiMeter            = global.MeterProvider().Meter("scrm_cti_meter")
	cleanOrdersCounter  syncint64.Counter
	closeProjectCounter syncint64.Counter
	pushOrdersCounter   syncint64.Counter
	pullCDRCounter      syncint64.Counter
)

func init() {
	coc, err := ctiMeter.SyncInt64().Counter(
		"cti.clean_orders.counter",
		instrument.WithUnit("1"),
		instrument.WithDescription("cleaned orders count"),
	)
	if err != nil {
		panic(err)
	}
	cleanOrdersCounter = coc

	cpc, err := ctiMeter.SyncInt64().Counter(
		"cti.close_project.counter",
		instrument.WithUnit("1"),
		instrument.WithDescription("closed project count"),
	)
	if err != nil {
		panic(err)
	}
	closeProjectCounter = cpc

	poc, err := ctiMeter.SyncInt64().Counter(
		"cti.push_orders.counter",
		instrument.WithUnit("1"),
		instrument.WithDescription("pushed orders count"),
	)
	if err != nil {
		panic(err)
	}
	pushOrdersCounter = poc

	pcc, err := ctiMeter.SyncInt64().Counter(
		"cti.pull_cdr.counter",
		instrument.WithUnit("1"),
		instrument.WithDescription("pull cdr count"),
	)
	if err != nil {
		panic(err)
	}
	pullCDRCounter = pcc
}
