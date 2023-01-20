package services

import (
	"eventsync/models"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestEventService_ExtractEventKey(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name         string
		args         args
		wantEventKey string
	}{
		{
			name: "ok",
			args: args{
				path: EventPathPrefix + "ok",
			},
			wantEventKey: "ok",
		},
		{
			name: "no pattern enforced",
			args: args{
				path: "any/" + "ok",
			},
			wantEventKey: "",
		},
		{
			name: "prefix to pattern",
			args: args{
				path: "any" + EventPathPrefix + "ok",
			},
			wantEventKey: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotEventKey := ExtractEventKey(tt.args.path); gotEventKey != tt.wantEventKey {
				t.Errorf("ExtractEventKey() = %v, wantErr %v", gotEventKey, tt.wantEventKey)
			}
		})
	}
}

func TestFormatEvent(t *testing.T) {
	content := "Hello, world!"
	body := io.NopCloser(strings.NewReader(content))
	queryParam := map[string][]string{
		"entry0": {},
		"entry1": {
			"string1",
		},
		"entry2": {
			"string1",
			"string2",
		},
	}
	header := map[string][]string{
		"entry0": {},
		"entry1": {
			"string1",
		},
		"entry2": {
			"string1",
			"string2",
		},
	}

	type args struct {
		eventKey   string
		queryParam map[string][]string
		headers    map[string][]string
		body       io.ReadCloser
		method     string
	}
	tests := []struct {
		name      string
		args      args
		wantEvent models.Event
		wantErr   bool
	}{
		{
			name: "validate copy",
			args: args{
				eventKey:   "entry1",
				queryParam: queryParam,
				headers:    header,
				body:       body,
				method:     "GET",
			},
			//Only validated field! Change the check test if more fields are required
			wantEvent: models.Event{
				EventKey:    "entry1",
				Headers:     header,
				QueryParams: queryParam,
				Content:     content,
				Method:      models.HttpMethodTypeGet,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEvent, err := FormatEvent(tt.args.eventKey, tt.args.queryParam, tt.args.headers, tt.args.body, tt.args.method)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotEvent.Headers, tt.wantEvent.Headers) ||
				!reflect.DeepEqual(gotEvent.QueryParams, tt.wantEvent.QueryParams) ||
				gotEvent.EventKey != tt.wantEvent.EventKey ||
				gotEvent.Content != tt.wantEvent.Content {
				t.Errorf("FormatEvent() gotEvent = %v, wantErr %v", gotEvent, tt.wantEvent)
			}
		})
	}
}

func TestEventService_MatchEndpoint(t *testing.T) {
	type fields struct {
		configService *ConfigService
	}
	type args struct {
		eventKeyValue string
		method        string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "ko empty",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: generateValidConfig(),
				},
			},
			args: args{
				eventKeyValue: "",
				method:        "GET",
			},
			wantErr: true,
		},
		{
			name: "ko no match",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: generateValidConfig(),
				},
			},
			args: args{
				eventKeyValue: "entry3",
				method:        "GET",
			},
			wantErr: true,
		},
		{
			name: "ko nil endpoint (should never occur)",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: func() *models.EventSyncConfig {
						e := generateValidConfig()
						e.Endpoints = nil
						return e
					}(),
				},
			},
			args: args{
				eventKeyValue: "entry1",
				method:        "GET",
			},
			wantErr: true,
		},
		{
			name: "ko no method (should never occur)",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: generateValidConfig(),
				},
			},
			args: args{
				eventKeyValue: "entry1",
				method:        "",
			},
			wantErr: true,
		},
		{
			name: "ko not accepted method",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: generateValidConfig(),
				},
			},
			args: args{
				eventKeyValue: "entry1",
				method:        "POST",
			},
			wantErr: true,
		},
		{
			name: "ko lower case accepted method",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: generateValidConfig(),
				},
			},
			args: args{
				eventKeyValue: "entry1",
				method:        "get",
			},
			wantErr: true,
		},
		{
			name: "ok 1st",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: generateValidConfig(),
				},
			},
			args: args{
				eventKeyValue: "entry1",
				method:        "GET",
			},
			wantErr: false,
		},
		{
			name: "ok 2nd",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: generateValidConfig(),
				},
			},
			args: args{
				eventKeyValue: "entry2",
				method:        "POST",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &EventService{
				configService: tt.fields.configService,
			}
			if got := e.MatchEndpoint(tt.args.eventKeyValue, tt.args.method); (got != nil) != tt.wantErr {
				t.Errorf("MatchEndpoint() = %v, wantErr %v", got, tt.wantErr)
			}
		})
	}
}

func TestEventService_checkTriggerConditions(t *testing.T) {
	type fields struct {
		configService *ConfigService
	}
	type args struct {
		eventList map[string]models.EventList
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		wantNeedTrigger bool
	}{
		{
			name: "ko missing 1 endpoint",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: generateValidConfig(),
				},
			},
			args: args{
				eventList: map[string]models.EventList{
					"entry1": {
						NumberOfEvents: 1,
					},
					"entry2": {
						NumberOfEvents: 0,
					},
				},
			},
			wantNeedTrigger: false,
		},
		{
			name: "ko missing 1 entry",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: generateValidConfig(),
				},
			},
			args: args{
				eventList: map[string]models.EventList{
					"entry1": {
						NumberOfEvents: 1,
					},
				},
			},
			wantNeedTrigger: false,
		},
		{
			name: "ko missing 1 event",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: func() *models.EventSyncConfig {
						e := generateValidConfig()
						e.Endpoints[0].MinNbOfOccurrence = 2
						return e
					}(),
				},
			},
			args: args{
				eventList: map[string]models.EventList{
					"entry1": {
						NumberOfEvents: 1,
					},
				},
			},
			wantNeedTrigger: false,
		},
		{
			name: "ok number of event 1",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: generateValidConfig(),
				},
			},
			args: args{
				eventList: map[string]models.EventList{
					"entry1": {
						NumberOfEvents: 1,
					},
					"entry2": {
						NumberOfEvents: 1,
					},
				},
			},
			wantNeedTrigger: true,
		},
		{
			name: "ok number of event 2",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: func() *models.EventSyncConfig {
						e := generateValidConfig()
						e.Endpoints[0].MinNbOfOccurrence = 2
						return e
					}(),
				},
			},
			args: args{
				eventList: map[string]models.EventList{
					"entry1": {
						NumberOfEvents: 2,
					},
					"entry2": {
						NumberOfEvents: 1,
					},
				},
			},
			wantNeedTrigger: true,
		},
		{
			name: "ok number of event > 2",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: func() *models.EventSyncConfig {
						e := generateValidConfig()
						e.Endpoints[0].MinNbOfOccurrence = 2
						return e
					}(),
				},
			},
			args: args{
				eventList: map[string]models.EventList{
					"entry1": {
						NumberOfEvents: 3,
					},
					"entry2": {
						NumberOfEvents: 1,
					},
				},
			},
			wantNeedTrigger: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &EventService{
				configService: tt.fields.configService,
			}
			if gotNeedTrigger := e.checkTriggerConditions(tt.args.eventList); gotNeedTrigger != tt.wantNeedTrigger {
				t.Errorf("checkTriggerConditions() = %v, wantErr %v", gotNeedTrigger, tt.wantNeedTrigger)
			}
		})
	}
}
