package service

import (
	"reflect"
	"testing"
)

func TestBatteryTriggerRoutesCombineDatasourcesByPoint(t *testing.T) {
	previousDatasources := batteryTriggerRoutes.byDatasource
	previousPoints := batteryTriggerRoutes.byPoint
	batteryTriggerRoutes = batteryTriggerRouteState{
		byDatasource: make(map[string]map[string]struct{}),
		byPoint:      make(map[string][]string),
	}
	t.Cleanup(func() {
		batteryTriggerRoutes = batteryTriggerRouteState{byDatasource: previousDatasources, byPoint: previousPoints}
	})

	replaceBatteryDatasourceRoutes("", map[string]struct{}{
		"865619085125863 01_point_a": {},
	})
	replaceBatteryDatasourceRoutes("renter2", map[string]struct{}{
		"865619085125863 01_point_a": {},
		"865619085125863 01_point_b": {},
	})

	if got, want := batteryTriggerDBNames("865619085125863", "01_point_a"), []string{"", "renter2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("routes = %#v, want %#v", got, want)
	}
	if got, want := batteryTriggerDBNames("865619085125863", "01_point_b"), []string{"renter2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("routes = %#v, want %#v", got, want)
	}
}

func TestFormatBatteryTriggerPointTextKeepsDatasourceInternal(t *testing.T) {
	if got := formatBatteryTriggerPointText("gw", "01_point", "1", 100, ""); got != "gw 01_point 1 100" {
		t.Fatalf("default body = %q", got)
	}
	if got := formatBatteryTriggerPointText("gw", "01_point", "1", 100, "renter2"); got != "gw 01_point 1 100 renter2" {
		t.Fatalf("renter body = %q", got)
	}
}

func TestRemoveInactiveBatteryDatasources(t *testing.T) {
	previousDatasources := batteryTriggerRoutes.byDatasource
	previousPoints := batteryTriggerRoutes.byPoint
	batteryTriggerRoutes = batteryTriggerRouteState{
		byDatasource: map[string]map[string]struct{}{
			"":        {"gw 01_a": {}},
			"renter2": {"gw 01_b": {}},
		},
		byPoint: make(map[string][]string),
	}
	rebuildBatteryPointRoutesLocked()
	t.Cleanup(func() {
		batteryTriggerRoutes = batteryTriggerRouteState{byDatasource: previousDatasources, byPoint: previousPoints}
	})

	if !removeInactiveBatteryDatasources(map[string]struct{}{"": {}}) {
		t.Fatal("expected route removal")
	}
	if got := batteryTriggerDBNames("gw", "01_b"); len(got) != 0 {
		t.Fatalf("inactive datasource routes remain: %#v", got)
	}
}
