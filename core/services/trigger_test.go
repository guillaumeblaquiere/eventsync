package services

import (
	"eventsync/models"
	"reflect"
	"testing"
	"time"
)

var (
	now    = time.Date(2022, 03, 28, 0, 0, 0, 0, time.UTC)
	before = now.Add(-1 * time.Hour)
	after  = now.Add(1 * time.Hour)
)

func generateEvents() (events map[string][]models.Event) {
	events = make(map[string][]models.Event)
	events["entry1"] = []models.Event{
		{
			FirestoreDocumentID: "id1",
			Datetime:            now,
		},
	}
	events["entry2"] = []models.Event{
		{
			FirestoreDocumentID: "id2.1",
			Datetime:            before,
		},
		{
			FirestoreDocumentID: "id2.2",
			Datetime:            now,
		},
		{
			FirestoreDocumentID: "id2.3",
			Datetime:            after,
		},
	}
	return
}

func generateExpectedEventList() (eventList map[string]*models.EventList) {

	eventList = map[string]*models.EventList{
		"entry1": {
			FirstEventDate:    &now,
			LastEventDate:     &now,
			NumberOfEvents:    1,
			MinNbOfOccurrence: generateValidConfig().Endpoints[0].MinNbOfOccurrence,
			EventToSend:       generateValidConfig().Endpoints[0].EventToSend,
			Events:            generateEvents()["entry1"],
		},
		"entry2": {
			FirstEventDate:    &before,
			LastEventDate:     &after,
			NumberOfEvents:    3,
			MinNbOfOccurrence: generateValidConfig().Endpoints[1].MinNbOfOccurrence,
			EventToSend:       generateValidConfig().Endpoints[1].EventToSend,
			Events:            generateEvents()["entry2"],
		},
	}
	return
}

func TestTriggerService_createEventGenerated(t1 *testing.T) {
	type fields struct {
		configService *ConfigService
	}
	type args struct {
		events map[string][]models.Event
	}
	tests := []struct {
		name               string
		fields             fields
		args               args
		wantEventGenerated models.EventGenerated
	}{
		{
			name: "empty list",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: generateValidConfig(),
				},
			},
			args: args{
				events: map[string][]models.Event{},
			},
			wantEventGenerated: models.EventGenerated{
				EventID:     "d41d8cd98f00b204e9800998ecf8427e",
				Date:        now,
				ServiceName: generateValidConfig().ServiceName,
				TriggerTpe:  generateValidConfig().Trigger.Type,
				Events: map[string]*models.EventList{
					"entry1": {
						MinNbOfOccurrence: generateValidConfig().Endpoints[0].MinNbOfOccurrence,
						EventToSend:       generateValidConfig().Endpoints[0].EventToSend,
					},
					"entry2": {
						MinNbOfOccurrence: generateValidConfig().Endpoints[1].MinNbOfOccurrence,
						EventToSend:       generateValidConfig().Endpoints[1].EventToSend,
					},
				},
			},
		},
		{
			name: "Base",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: generateValidConfig(),
				},
			},
			args: args{
				events: generateEvents(),
			},
			wantEventGenerated: models.EventGenerated{
				EventID:     "5ad5318b7a774c084bad1d649ac0eb90",
				Date:        now,
				ServiceName: generateValidConfig().ServiceName,
				TriggerTpe:  generateValidConfig().Trigger.Type,
				Events:      generateExpectedEventList(),
			},
		},
		{
			name: "Only first",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: func() *models.EventSyncConfig {
						g := generateValidConfig()
						g.Endpoints[0].EventToSend = models.EventToSendTypeFirst
						g.Endpoints[1].EventToSend = models.EventToSendTypeFirst
						return g
					}(),
				},
			},
			args: args{
				events: generateEvents(),
			},
			wantEventGenerated: models.EventGenerated{
				EventID:     "d24f5dca0b6a5633a2b53f6ff7d82299",
				Date:        now,
				ServiceName: generateValidConfig().ServiceName,
				TriggerTpe:  generateValidConfig().Trigger.Type,
				Events: func() map[string]*models.EventList {
					g := generateExpectedEventList()
					g["entry1"].EventToSend = models.EventToSendTypeFirst

					g["entry2"].Events = []models.Event{
						g["entry2"].Events[0],
					}
					g["entry2"].EventToSend = models.EventToSendTypeFirst

					return g
				}(),
			},
		},
		{
			name: "Only last",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: func() *models.EventSyncConfig {
						g := generateValidConfig()
						g.Endpoints[0].EventToSend = models.EventToSendTypeLast
						g.Endpoints[1].EventToSend = models.EventToSendTypeLast
						return g
					}(),
				},
			},
			args: args{
				events: generateEvents(),
			},
			wantEventGenerated: models.EventGenerated{
				EventID:     "6d52da24f61dda7b9b5ac9ad5ebea7b0",
				Date:        now,
				ServiceName: generateValidConfig().ServiceName,
				TriggerTpe:  generateValidConfig().Trigger.Type,
				Events: func() map[string]*models.EventList {
					g := generateExpectedEventList()
					g["entry1"].EventToSend = models.EventToSendTypeLast

					g["entry2"].Events = []models.Event{
						g["entry2"].Events[2],
					}
					g["entry2"].EventToSend = models.EventToSendTypeLast

					return g
				}(),
			},
		},
		{
			name: "Only boundaries",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: func() *models.EventSyncConfig {
						g := generateValidConfig()
						g.Endpoints[0].EventToSend = models.EventToSendTypeBoundaries
						g.Endpoints[1].EventToSend = models.EventToSendTypeBoundaries
						return g
					}(),
				},
			},
			args: args{
				events: generateEvents(),
			},
			wantEventGenerated: models.EventGenerated{
				EventID:     "ca92803edad18d2bbf3da3ea90a33ead",
				Date:        now,
				ServiceName: generateValidConfig().ServiceName,
				TriggerTpe:  generateValidConfig().Trigger.Type,
				Events: func() map[string]*models.EventList {
					g := generateExpectedEventList()
					g["entry1"].EventToSend = models.EventToSendTypeBoundaries

					g["entry2"].Events = []models.Event{
						g["entry2"].Events[0],
						g["entry2"].Events[2],
					}
					g["entry2"].EventToSend = models.EventToSendTypeBoundaries

					return g
				}(),
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TriggerService{
				configService: tt.fields.configService,
			}
			gotEventGenerated := t.createEventGenerated(tt.args.events)
			gotEventGenerated.Date = now
			if gotEventGenerated.ServiceName != tt.wantEventGenerated.ServiceName ||
				gotEventGenerated.Date != tt.wantEventGenerated.Date ||
				gotEventGenerated.TriggerTpe != tt.wantEventGenerated.TriggerTpe ||
				gotEventGenerated.EventID != tt.wantEventGenerated.EventID {
				t1.Errorf("createEventGenerated() = %+v, want %+v", gotEventGenerated, tt.wantEventGenerated)
			} else {
				//deep equals in the map entries
				for eventKey, eventList := range gotEventGenerated.Events {
					if !reflect.DeepEqual(eventList, tt.wantEventGenerated.Events[eventKey]) {
						t1.Errorf("createEventGenerated().eventList[%s] = %+v, want %+v", eventKey, eventList, tt.wantEventGenerated.Events[eventKey])
					}
				}
				if len(tt.wantEventGenerated.Events) > len(gotEventGenerated.Events) {
					t1.Errorf("len(createEventGenerated().Event) = %d, want %d", len(gotEventGenerated.Events), len(tt.wantEventGenerated.Events))
				}
			}
		})
	}
}
