package services

import (
	"eventsync/models"
	"testing"
)

func generateEventlist() (eventList map[string]models.EventList) {
	eventList = make(map[string]models.EventList)
	eventList["entry1"] = models.EventList{
		NumberOfEvents: 1,
		Events: []models.Event{
			{
				FirestoreDocumentID: "id1",
			},
		},
	}
	eventList["entry2"] = models.EventList{
		NumberOfEvents: 1,
		Events: []models.Event{
			{
				FirestoreDocumentID: "id2",
			},
		},
	}
	return
}

func Test_generateEventID(t *testing.T) {
	type args struct {
		eventList map[string]models.EventList
	}
	tests := []struct {
		name        string
		args        args
		wantEventId string
	}{
		{
			name:        "OK empty list",
			args:        args{},
			wantEventId: "d41d8cd98f00b204e9800998ecf8427e",
		},
		{
			name: "OK full list",
			args: args{
				eventList: generateEventlist(),
			},
			wantEventId: "a313af57d0cda095939730385b3ee84c",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotEventId := generateEventID(tt.args.eventList); gotEventId != tt.wantEventId {
				t.Errorf("generateEventID() = %v, want %v", gotEventId, tt.wantEventId)
			}
		})
	}
}
