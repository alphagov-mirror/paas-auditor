package eventstore_test

import (
	"encoding/json"
	"time"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/alphagov/paas-billing/testenv"
	uuid "github.com/satori/go.uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {

	var (
		cfg eventstore.Config
	)

	BeforeEach(func() {
		cfg = testenv.BasicConfig
	})

	It("should be idempotent", func() {
		db, err := testenv.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()
		Expect(db.Schema.Init()).To(Succeed())
		Expect(db.Schema.Init()).To(Succeed())
	})

	It("should normalize *_usage_events tables into a consistant format with durations", func() {
		cfg.AddPlan(eventstore.PricingPlan{
			PlanGUID:  eventstore.ComputePlanGUID,
			ValidFrom: "2001-01-01",
			Name:      "APP_PLAN_1",
			Components: []eventstore.PricingPlanComponent{
				{
					Name:         "compute",
					Formula:      "ceil($time_in_seconds/3600) * 0.01",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})
		cfg.AddPlan(eventstore.PricingPlan{
			PlanGUID:  "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
			ValidFrom: "2001-01-01",
			Name:      "DB_PLAN_1",
			Components: []eventstore.PricingPlanComponent{
				{
					Name:         "compute",
					Formula:      "ceil($time_in_seconds/3600) * 1",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})

		app1EventStart := testenv.Row{
			"guid":        "ee28a570-f485-48e1-87d0-98b7b8b66dfa",
			"created_at":  "2001-01-01T00:00Z",
			"raw_message": json.RawMessage(`{"state": "STARTED", "app_guid": "c85e98f0-6d1b-4f45-9368-ea58263165a0", "app_name": "APP1", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "ORG1-SPACE1", "process_type": "web", "instance_count": 1, "previous_state": "STARTED", "memory_in_mb_per_instance": 1024}`),
		}
		app1EventStop := testenv.Row{
			"guid":        "8d9036c5-8367-497d-bb56-94bfcac6621a",
			"created_at":  "2001-01-01T01:00Z",
			"raw_message": json.RawMessage(`{"state": "STOPPED", "app_guid": "c85e98f0-6d1b-4f45-9368-ea58263165a0", "app_name": "APP1", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "ORG1-SPACE1", "process_type": "web", "instance_count": 1, "previous_state": "STARTED", "memory_in_mb_per_instance": 1024}`),
		}
		service1EventStart := testenv.Row{
			"guid":        "c497eb13-f48a-4859-be53-5569f302b516",
			"created_at":  "2001-01-01T00:00Z",
			"raw_message": json.RawMessage(`{"state": "CREATED", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "bd405d91-0b7c-4b8c-96ef-8b4c1e26e75d", "space_name": "sandbox", "service_guid": "efadb775-58c4-4e17-8087-6d0f4febc489", "service_label": "postgres", "service_plan_guid": "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5", "service_plan_name": "Free", "service_instance_guid": "f3f98365-6a95-4bbd-ab8f-527a7957a41f", "service_instance_name": "ja-rails-postgres", "service_instance_type": "managed_service_instance"}`),
		}
		service1EventStop := testenv.Row{
			"guid":        "6d52b4f4-9e33-4504-8fca-fd9e33af11a6",
			"created_at":  "2001-01-01T01:00Z",
			"raw_message": json.RawMessage(`{"state": "DELETED", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "bd405d91-0b7c-4b8c-96ef-8b4c1e26e75d", "space_name": "sandbox", "service_guid": "efadb775-58c4-4e17-8087-6d0f4febc489", "service_label": "postgres", "service_plan_guid": "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5", "service_plan_name": "Free", "service_instance_guid": "f3f98365-6a95-4bbd-ab8f-527a7957a41f", "service_instance_name": "ja-rails-postgres", "service_instance_type": "managed_service_instance"}`),
		}

		db, err := testenv.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		Expect(db.Insert("app_usage_events", app1EventStart, app1EventStop)).To(Succeed())
		Expect(db.Insert("service_usage_events", service1EventStart, service1EventStop)).To(Succeed())
		Expect(db.Schema.Refresh()).To(Succeed())

		Expect(
			db.Query(`select * from events`),
		).To(MatchJSON(testenv.Rows{
			{
				"duration":        "[\"2001-01-01 00:00:00+00\",\"2001-01-01 01:00:00+00\")",
				"event_guid":      "ee28a570-f485-48e1-87d0-98b7b8b66dfa",
				"memory_in_mb":    1024,
				"number_of_nodes": 1,
				"org_guid":        "51ba75ef-edc0-47ad-a633-a8f6e8770944",
				"plan_guid":       "f4d4b95a-f55e-4593-8d54-3364c25798c4",
				"plan_name":       "app",
				"resource_guid":   "c85e98f0-6d1b-4f45-9368-ea58263165a0",
				"resource_name":   "APP1",
				"resource_type":   "app",
				"space_guid":      "276f4886-ac40-492d-a8cd-b2646637ba76",
				"storage_in_mb":   0,
			},
			{
				"duration":        "[\"2001-01-01 00:00:00+00\",\"2001-01-01 01:00:00+00\")",
				"event_guid":      "c497eb13-f48a-4859-be53-5569f302b516",
				"memory_in_mb":    nil,
				"number_of_nodes": nil,
				"org_guid":        "51ba75ef-edc0-47ad-a633-a8f6e8770944",
				"plan_guid":       "efb5f1ce-0a8a-435d-a8b2-6b2b61c6dbe5",
				"plan_name":       "Free",
				"resource_guid":   "f3f98365-6a95-4bbd-ab8f-527a7957a41f",
				"resource_name":   "ja-rails-postgres",
				"resource_type":   "postgres",
				"space_guid":      "bd405d91-0b7c-4b8c-96ef-8b4c1e26e75d",
				"storage_in_mb":   nil,
			},
		}))
	})

	It("only outputs a single resource row because the others have zero duration", func() {
		cfg.AddPlan(eventstore.PricingPlan{
			PlanGUID:  eventstore.ComputePlanGUID,
			ValidFrom: "2001-01-01",
			Name:      "APP_PLAN_1",
			Components: []eventstore.PricingPlanComponent{
				{
					Name:         "compute",
					Formula:      "ceil($time_in_seconds/3600) * 0.01",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})

		db, err := testenv.Open(cfg)
		Expect(err).ToNot(HaveOccurred())
		defer db.Close()

		Expect(db.Insert("app_usage_events",
			testenv.Row{
				"guid":        "ee28a570-f485-48e1-87d0-98b7b8b66dfa",
				"created_at":  "2001-01-01T00:00Z",
				"raw_message": json.RawMessage(`{"state": "STARTED", "app_guid": "c85e98f0-6d1b-4f45-9368-ea58263165a0", "app_name": "APP1", "org_guid": "51ba75ef-edc0-47ad-a633-a8f6e8770944", "space_guid": "276f4886-ac40-492d-a8cd-b2646637ba76", "space_name": "ORG1-SPACE1", "process_type": "web", "instance_count": 1, "previous_state": "STARTED", "memory_in_mb_per_instance": 1024}`),
			},
		)).To(Succeed())

		Expect(db.Schema.Refresh()).To(Succeed())

		Expect(db.Get(`SELECT COUNT(*) FROM billable_event_components`)).To(BeNumerically("==", 1))
	})

	It("should ensure plan has unique plan_guid + valid_from", func() {
		cfg.AddPlan(eventstore.PricingPlan{
			PlanGUID:  eventstore.ComputePlanGUID,
			ValidFrom: "2001-01-01",
			Name:      "APP_PLAN_1",
			Components: []eventstore.PricingPlanComponent{
				{
					Name:         "compute",
					Formula:      "ceil($time_in_seconds/3600) * 0.01",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})
		cfg.AddPlan(eventstore.PricingPlan{
			PlanGUID:  eventstore.ComputePlanGUID,
			ValidFrom: "2001-01-01",
			Name:      "APP_PLAN_1",
			Components: []eventstore.PricingPlanComponent{
				{
					Name:         "compute",
					Formula:      "ceil($time_in_seconds/3600) * 0.01",
					CurrencyCode: "GBP",
					VATCode:      "Standard",
				},
			},
		})
		db, err := testenv.Open(cfg)
		Expect(err).To(MatchError(ContainSubstring(`violates unique constraint`)))
		if err == nil {
			db.Close()
		}
	})

	DescribeTable("reject placing plans with valid_from that isn't the start of the month",
		func(timestamp string) {
			db, err := testenv.Open(eventstore.Config{
				PricingPlans: []eventstore.PricingPlan{
					{
						PlanGUID:  uuid.NewV4().String(),
						ValidFrom: timestamp,
						Name:      "bad-plan",
						Components: []eventstore.PricingPlanComponent{
							{
								Name:         "compute",
								Formula:      "1",
								CurrencyCode: "GBP",
								VATCode:      "Standard",
							},
						},
					},
				},
			})
			if err == nil {
				db.Close()
			}
			Expect(err).To(MatchError(ContainSubstring(`violates check constraint "valid_from_start_of_month"`)))
		},
		Entry("not first day of month", "2017-04-04T00:00:00Z"),
		Entry("not midnight (hour)", "2017-04-01T01:00:00Z"),
		Entry("not midnight (minute)", "2017-04-01T00:01:00Z"),
		Entry("not midnight (second)", "2017-04-01T01:00:01Z"),
		Entry("not midnight (different timezone)", "2017-04-01T00:00:00+01:00"),
	)

	DescribeTable("reject vat_rates with valid_from that isn't the start of the month",
		func(timestamp string) {
			db, err := testenv.Open(eventstore.Config{
				VATRates: []eventstore.VATRate{
					{
						ValidFrom: timestamp,
						Code:      "Standard",
						Rate:      0,
					},
				},
			})
			if err == nil {
				db.Close()
			}
			Expect(err).To(MatchError(ContainSubstring(`violates check constraint "valid_from_start_of_month"`)))
		},
		Entry("not first day of month", "2017-04-04T00:00:00Z"),
		Entry("not midnight (hour)", "2017-04-01T01:00:00Z"),
		Entry("not midnight (minute)", "2017-04-01T00:01:00Z"),
		Entry("not midnight (second)", "2017-04-01T01:00:01Z"),
		Entry("not midnight (different timezone)", "2017-04-01T00:00:00+01:00"),
	)

	DescribeTable("reject currency_rates with valid_from that isn't the start of the month",
		func(timestamp string) {
			db, err := testenv.Open(eventstore.Config{
				CurrencyRates: []eventstore.CurrencyRate{
					{
						ValidFrom: timestamp,
						Code:      "USD",
						Rate:      0.8,
					},
				},
			})
			if err == nil {
				db.Close()
			}
			Expect(err).To(MatchError(ContainSubstring(`violates check constraint "valid_from_start_of_month"`)))
		},
		Entry("not first day of month", "2017-04-04T00:00:00Z"),
		Entry("not midnight (hour)", "2017-04-01T01:00:00Z"),
		Entry("not midnight (minute)", "2017-04-01T00:01:00Z"),
		Entry("not midnight (second)", "2017-04-01T01:00:01Z"),
		Entry("not midnight (different timezone)", "2017-04-01T00:00:00+01:00"),
	)

	DescribeTable("allow whitelisted currency codes",
		func(code string) {
			db, err := testenv.Open(eventstore.Config{
				CurrencyRates: []eventstore.CurrencyRate{
					{
						ValidFrom: "2001-01-01",
						Code:      code,
						Rate:      0.8,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			if err == nil {
				defer db.Close()
			}
		},
		Entry("£ UK Sterling", "GBP"),
		Entry("$ US Dollar", "USD"),
		Entry("€ Euro", "EUR"),
	)

	DescribeTable("reject unknown currency_codes",
		func(code string) {
			db, err := testenv.Open(eventstore.Config{
				CurrencyRates: []eventstore.CurrencyRate{
					{
						ValidFrom: "2001-01-01",
						Code:      code,
						Rate:      0.8,
					},
				},
			})
			if err == nil {
				db.Close()
			}
			Expect(err).To(MatchError(ContainSubstring(`invalid currency rate: invalid input value for enum currency_code`)))
		},
		Entry("no lowercase", "usd"),
		Entry("no symbols", "$"),
		Entry("no random codes", "UKP"),
		Entry("no unknown", "XXX"),
	)

	DescribeTable("allow whitelisted vat_rates",
		func(code string) {
			db, err := testenv.Open(eventstore.Config{
				VATRates: []eventstore.VATRate{
					{
						ValidFrom: "2001-01-01",
						Code:      code,
						Rate:      0.1,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			if err == nil {
				defer db.Close()
			}
		},
		Entry("allow: Standard", "Standard"),
		Entry("allow: Reduced", "Reduced"),
		Entry("allow: Zero", "Zero"),
	)

	DescribeTable("reject unknown vat_rates",
		func(code string) {
			db, err := testenv.Open(eventstore.Config{
				VATRates: []eventstore.VATRate{
					{
						ValidFrom: "2001-01-01",
						Code:      code,
						Rate:      0.8,
					},
				},
			})
			if err == nil {
				db.Close()
			}
			Expect(err).To(MatchError(ContainSubstring(`invalid vat rate: invalid input value for enum vat_code`)))
		},
		Entry("no lowercase", "standard"),
		Entry("no uppercase", "ZERO"),
		Entry("no random codes", "myrate"),
	)

	DescribeTable("should store events of difference kinds",
		func(kind string) {
			db, err := testenv.Open(eventstore.Config{})
			Expect(err).ToNot(HaveOccurred())
			defer db.Close()
			event1 := eventio.RawEvent{
				GUID:       "94147a2f-2626-4445-8b4e-22ebe8071a29",
				CreatedAt:  time.Date(2001, 1, 1, 1, 1, 1, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-1"}`),
			}
			event2 := eventio.RawEvent{
				GUID:       "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
				CreatedAt:  time.Date(2002, 2, 2, 2, 2, 2, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-2"}`),
			}
			event3 := eventio.RawEvent{
				GUID:       "395b7d4c-c859-4a28-9a53-6b15fab447c7",
				CreatedAt:  time.Date(2003, 3, 3, 3, 3, 3, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-3"}`),
			}
			By("attempting to store a batch of events", func() {
				err := db.Schema.StoreEvents([]eventio.RawEvent{
					event1,
					event2,
					event3,
				})
				Expect(err).ToNot(HaveOccurred())
			})
			By("fetching the stored events back", func() {
				storedEvents, err := db.Schema.GetEvents(eventio.RawEventFilter{
					Kind: kind,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(storedEvents).To(Equal([]eventio.RawEvent{
					event3,
					event2,
					event1,
				}))
			})
		},
		Entry("app usage event", "app"),
		Entry("service usage event", "service"),
		Entry("compose event", "compose"),
	)

	DescribeTable("should not commit when batch contains invalid app event",
		func(kind string, expectedErr string, badEvent eventio.RawEvent) {
			db, err := testenv.Open(eventstore.Config{})
			Expect(err).ToNot(HaveOccurred())
			defer db.Close()
			event1 := eventio.RawEvent{
				GUID:       "94147a2f-2626-4445-8b4e-22ebe8071a29",
				CreatedAt:  time.Date(2002, 2, 2, 2, 2, 2, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-2"}`),
			}
			event3 := eventio.RawEvent{
				GUID:       "395b7d4c-c859-4a28-9a53-6b15fab447c7",
				CreatedAt:  time.Date(2003, 3, 3, 3, 3, 3, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-3"}`),
			}
			By("attempting to store a bad batch of events", func() {
				err := db.Schema.StoreEvents([]eventio.RawEvent{
					event1,
					badEvent,
					event3,
				})
				Expect(err).To(MatchError(ContainSubstring(expectedErr)))
			})
			By("fetching no events back", func() {
				storedEvents, err := db.Schema.GetEvents(eventio.RawEventFilter{
					Kind: kind,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(storedEvents).To(Equal([]eventio.RawEvent{}))
			})
		},
		Entry("app event with no GUID", "app", "must have a GUID", eventio.RawEvent{
			CreatedAt:  time.Date(2002, 1, 1, 1, 1, 1, 0, time.UTC),
			Kind:       "app",
			RawMessage: json.RawMessage(`{"name": "bad-app-2"}`),
		}),
		Entry("app event with no CreatedAt", "app", "must have a CreatedAt", eventio.RawEvent{
			GUID:       "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
			Kind:       "app",
			RawMessage: json.RawMessage(`{"name": "bad-app-2"}`),
		}),
		Entry("app event with no Kind", "app", "must have a Kind", eventio.RawEvent{
			GUID:       "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
			CreatedAt:  time.Date(2002, 1, 1, 1, 1, 1, 0, time.UTC),
			RawMessage: json.RawMessage(`{"name": "bad-app-2"}`),
		}),
		Entry("app event with no RawMessage", "app", "must have a RawMessage payload", eventio.RawEvent{
			GUID:      "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
			CreatedAt: time.Date(2002, 1, 1, 1, 1, 1, 0, time.UTC),
			Kind:      "app",
		}),
		Entry("compose event with no GUID", "compose", "must have a GUID", eventio.RawEvent{
			CreatedAt:  time.Date(2002, 1, 1, 1, 1, 1, 0, time.UTC),
			Kind:       "compose",
			RawMessage: json.RawMessage(`{"name": "bad-app-2"}`),
		}),
		Entry("compose event with no CreatedAt", "compose", "must have a CreatedAt", eventio.RawEvent{
			GUID:       "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
			Kind:       "compose",
			RawMessage: json.RawMessage(`{"name": "bad-app-2"}`),
		}),
		Entry("compose event with no Kind", "compose", "must have a Kind", eventio.RawEvent{
			GUID:       "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
			CreatedAt:  time.Date(2002, 1, 1, 1, 1, 1, 0, time.UTC),
			RawMessage: json.RawMessage(`{"name": "bad-app-2"}`),
		}),
		Entry("compose event with no RawMessage", "compose", "must have a RawMessage payload", eventio.RawEvent{
			GUID:      "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
			CreatedAt: time.Date(2002, 1, 1, 1, 1, 1, 0, time.UTC),
			Kind:      "compose",
		}),
	)

	DescribeTable("should be an error to GetEvents with invalid Kind",
		func(kind string, expectedErr string) {
			db, err := testenv.Open(eventstore.Config{})
			Expect(err).ToNot(HaveOccurred())
			defer db.Close()
			storedEvents, err := db.Schema.GetEvents(eventio.RawEventFilter{
				Kind: kind,
			})
			Expect(err).To(MatchError(ContainSubstring(expectedErr)))
			Expect(storedEvents).To(BeNil())
		},
		Entry("unset kind", "", "you must supply a kind to filter events by"),
		Entry("unknown kind", "unknown", "cannot query events of kind 'unknown'"),
	)

	DescribeTable("should ignore events that already exist in the database",
		func(kind string) {
			db, err := testenv.Open(eventstore.Config{})
			Expect(err).ToNot(HaveOccurred())
			defer db.Close()
			event1 := eventio.RawEvent{
				GUID:       "94147a2f-2626-4445-8b4e-22ebe8071a29",
				CreatedAt:  time.Date(2001, 1, 1, 1, 1, 1, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-1"}`),
			}
			event2 := eventio.RawEvent{
				GUID:       "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
				CreatedAt:  time.Date(2002, 2, 2, 2, 2, 2, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-2"}`),
			}
			event3 := eventio.RawEvent{
				GUID:       "395b7d4c-c859-4a28-9a53-6b15fab447c7",
				CreatedAt:  time.Date(2003, 3, 3, 3, 3, 3, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-3"}`),
			}
			By("inserting a batch of events", func() {
				err := db.Schema.StoreEvents([]eventio.RawEvent{
					event1,
					event2,
				})
				Expect(err).ToNot(HaveOccurred())
			})
			By("inserting new same batch again", func() {
				err := db.Schema.StoreEvents([]eventio.RawEvent{
					event1,
					event2,
					event3,
				})
				Expect(err).ToNot(HaveOccurred())
			})
			By("fetching all events back", func() {
				storedEvents, err := db.Schema.GetEvents(eventio.RawEventFilter{
					Kind: kind,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(storedEvents).To(Equal([]eventio.RawEvent{
					event3,
					event2,
					event1,
				}))
			})
		},
		Entry("app event", "app"),
		Entry("service event", "service"),
		Entry("compose event", "compose"),
	)

	DescribeTable("should be able to fetch only the LAST known event",
		func(kind string) {
			db, err := testenv.Open(eventstore.Config{})
			Expect(err).ToNot(HaveOccurred())
			defer db.Close()
			event1 := eventio.RawEvent{
				GUID:       "94147a2f-2626-4445-8b4e-22ebe8071a29",
				CreatedAt:  time.Date(2001, 1, 1, 1, 1, 1, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-1"}`),
			}
			event2 := eventio.RawEvent{
				GUID:       "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
				CreatedAt:  time.Date(2003, 3, 3, 3, 3, 3, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-2"}`),
			}
			event3 := eventio.RawEvent{
				GUID:       "395b7d4c-c859-4a28-9a53-6b15fab447c7",
				CreatedAt:  time.Date(2002, 2, 2, 2, 2, 2, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-3"}`),
			}
			By("inserting a batch of events", func() {
				err := db.Schema.StoreEvents([]eventio.RawEvent{
					event1,
					event2,
					event3,
				})
				Expect(err).ToNot(HaveOccurred())
			})
			By("fetching back a single event", func() {
				storedEvents, err := db.Schema.GetEvents(eventio.RawEventFilter{
					Kind:  kind,
					Limit: 1,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(storedEvents).To(Equal([]eventio.RawEvent{
					event3,
				}))
			})
		},
		Entry("app event", "app"),
		Entry("service event", "service"),
		Entry("compose event", "compose"),
	)

	DescribeTable("should be able to fetch only the FIRST known event",
		func(kind string) {
			db, err := testenv.Open(eventstore.Config{})
			Expect(err).ToNot(HaveOccurred())
			defer db.Close()
			event1 := eventio.RawEvent{
				GUID:       "94147a2f-2626-4445-8b4e-22ebe8071a29",
				CreatedAt:  time.Date(2001, 1, 1, 1, 1, 1, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-1"}`),
			}
			event2 := eventio.RawEvent{
				GUID:       "7311ecc5-33f7-42f5-92b6-7f0789bf92a5",
				CreatedAt:  time.Date(2003, 3, 3, 3, 3, 3, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-2"}`),
			}
			event3 := eventio.RawEvent{
				GUID:       "395b7d4c-c859-4a28-9a53-6b15fab447c7",
				CreatedAt:  time.Date(2002, 2, 2, 2, 2, 2, 0, time.UTC),
				Kind:       kind,
				RawMessage: json.RawMessage(`{"name": "app-3"}`),
			}
			By("inserting a batch of events", func() {
				err := db.Schema.StoreEvents([]eventio.RawEvent{
					event1,
					event2,
					event3,
				})
				Expect(err).ToNot(HaveOccurred())
			})
			By("fetching back a single event", func() {
				storedEvents, err := db.Schema.GetEvents(eventio.RawEventFilter{
					Kind:    kind,
					Reverse: true,
					Limit:   1,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(storedEvents).To(Equal([]eventio.RawEvent{
					event1,
				}))
			})
		},
		Entry("app event", "app"),
		Entry("service event", "service"),
		Entry("compose event", "compose"),
	)

})