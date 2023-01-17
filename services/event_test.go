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
				t.Errorf("ExtractEventKey() = %v, want %v", gotEventKey, tt.wantEventKey)
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
			},
			//Only validated field! Change the check test if more fields are required
			wantEvent: models.Event{
				EventKey:    "entry1",
				Headers:     header,
				QueryParams: queryParam,
				Content:     content,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEvent, err := FormatEvent(tt.args.eventKey, tt.args.queryParam, tt.args.headers, tt.args.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotEvent.Headers, tt.wantEvent.Headers) ||
				!reflect.DeepEqual(gotEvent.QueryParams, tt.wantEvent.QueryParams) ||
				gotEvent.EventKey != tt.wantEvent.EventKey ||
				gotEvent.Content != tt.wantEvent.Content {
				t.Errorf("FormatEvent() gotEvent = %v, want %v", gotEvent, tt.wantEvent)
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
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
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
			},
			want: false,
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
			},
			want: false,
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
			},
			want: false,
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
			},
			want: true,
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
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &EventService{
				configService: tt.fields.configService,
			}
			if got := e.MatchEndpoint(tt.args.eventKeyValue); got != tt.want {
				t.Errorf("MatchEndpoint() = %v, want %v", got, tt.want)
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
			name: "ko missing 1 event",
			fields: fields{
				configService: &ConfigService{
					eventSyncConfig: generateValidConfig(),
				},
			},
			args: args{
				eventList: map[string]models.EventList{
					"entry1": models.EventList{
						NumberOfEvents: 1,
					},
					"entry2": models.EventList{
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
					"entry1": models.EventList{
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
					"entry1": models.EventList{
						NumberOfEvents: 1,
					},
					"entry2": models.EventList{
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
				t.Errorf("checkTriggerConditions() = %v, want %v", gotNeedTrigger, tt.wantNeedTrigger)
			}
		})
	}
}
