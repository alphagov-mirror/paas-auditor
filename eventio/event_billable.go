package eventio

import (
	"context"
	"encoding/json"
	"fmt"
)

type BillableEventReader interface {
	GetBillableEventRows(ctx context.Context, filter EventFilter) (BillableEventRows, error)
	GetBillableEvents(filter EventFilter) ([]BillableEvent, error)
}

type ConsolidatedBillableEventReader interface {
	GetConsolidatedBillableEventRows(ctx context.Context, filter EventFilter) (BillableEventRows, error)
	GetConsolidatedBillableEvents(filter EventFilter) ([]BillableEvent, error)
	IsRangeConsolidated(filter EventFilter) (bool, error)
}

type BillableEventConsolidator interface {
	ConsolidateAll() error
	ConsolidateFullMonths(startAt string, endAt string) error
	Consolidate(filter EventFilter) error
}

type BillableEventForecaster interface {
	ForecastBillableEventRows(ctx context.Context, events []UsageEvent, filter EventFilter) (BillableEventRows, error)
	ForecastBillableEvents(events []UsageEvent, filter EventFilter) ([]BillableEvent, error)
}

type BillableEventRows interface {
	Next() bool
	Close() error
	Err() error
	EventJSON() ([]byte, error)
	Event() (*BillableEvent, error)
}

type PriceComponent struct {
	Name         string `json:"name"`
	PlanName     string `json:"plan_name"`
	Start        string `json:"start"`
	Stop         string `json:"stop"`
	VatRate      string `json:"vat_rate"`
	VatCode      string `json:"vat_code"`
	CurrencyCode string `json:"currency_code"`
	CurrencyRate string `json:"currency_rate"`
	IncVAT       string `json:"inc_vat"`
	ExVAT        string `json:"ex_vat"`
}

type Price struct {
	IncVAT  string           `json:"inc_vat"`
	ExVAT   string           `json:"ex_vat"`
	Details []PriceComponent `json:"details"`
}

type BillableEvent struct {
	EventGUID     string `json:"event_guid"`
	EventStart    string `json:"event_start"`
	EventStop     string `json:"event_stop"`
	ResourceGUID  string `json:"resource_guid"`
	ResourceName  string `json:"resource_name"`
	ResourceType  string `json:"resource_type"`
	OrgGUID       string `json:"org_guid"`
	OrgName       string `json:"org_name"`
	SpaceGUID     string `json:"space_guid"`
	SpaceName     string `json:"space_name"`
	PlanGUID      string `json:"plan_guid"`
	PlanName      string `json:"plan_name"`
	ServiceGUID   string `json:"service_guid"`
	ServiceName   string `json:"service_name"`
	NumberOfNodes int64  `json:"number_of_nodes"`
	MemoryInMB    int64  `json:"memory_in_mb"`
	StorageInMB   int64  `json:"storage_in_mb"`
	Price         Price  `json:"price"`
}

func (e *BillableEvent) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("cannot Scan into BillableEvent with: %T", src)
	}
	if err := json.Unmarshal(source, e); err != nil {
		return err
	}
	return nil
}
