package services

import (
	"eventsync/models"
	"testing"
)

func generateValidConfig() *models.EventSyncConfig {
	return &models.EventSyncConfig{
		ServiceName: "myTest",
		Endpoints: []models.Endpoint{
			{
				EventKey: "entry1",
			},
			{
				EventKey: "entry2",
			},
		},
		Trigger: &models.Trigger{
			Type:                  models.TriggerTypeWindow,
			ObservationPeriod:     3600,
			KeepEventAfterTrigger: false,
		},
		TargetPubSub: &models.TargetPubSub{
			Topic: "projects/project123/topics/eventsync",
		},
	}
}

func TestConfigService_CheckConfig(t *testing.T) {
	type fields struct {
		eventSyncConfig *models.EventSyncConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "get error",
			fields: fields{
				eventSyncConfig: &models.EventSyncConfig{},
			},
			wantErr: true,
		},
		{
			name: "no error",
			fields: fields{
				eventSyncConfig: generateValidConfig(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ConfigService{
				eventSyncConfig: tt.fields.eventSyncConfig,
			}
			if err := c.CheckConfig(); (err != nil) != tt.wantErr {
				t.Errorf("CheckConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigService_checkConfigEndpoints(t *testing.T) {
	type fields struct {
		eventSyncConfig *models.EventSyncConfig
	}
	type args struct {
		logKO string
		logOK string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "with error nil endpoint",
			fields: fields{
				eventSyncConfig: func() *models.EventSyncConfig {
					e := generateValidConfig()
					e.Endpoints = nil
					return e
				}(),
			},
			args:    args{},
			wantErr: true,
		},
		{
			name: "with error empty string eventKey",
			fields: fields{
				eventSyncConfig: func() *models.EventSyncConfig {
					e := generateValidConfig()
					e.Endpoints[0].EventKey = ""
					return e
				}(),
			},
			args:    args{},
			wantErr: true,
		},
		{
			name: "with error duplicate eventKey",
			fields: fields{
				eventSyncConfig: func() *models.EventSyncConfig {
					e := generateValidConfig()
					e.Endpoints[0].EventKey = "duplicated"
					e.Endpoints[1].EventKey = "duplicated"
					return e
				}(),
			},
			args:    args{},
			wantErr: true,
		},
		{
			name: "with error 1 endpoint",
			fields: fields{
				eventSyncConfig: func() *models.EventSyncConfig {
					e := generateValidConfig()
					e.Endpoints = []models.Endpoint{
						e.Endpoints[0],
					}
					return e
				}(),
			},
			args:    args{},
			wantErr: true,
		},
		{
			name: "ok",
			fields: fields{
				eventSyncConfig: generateValidConfig(),
			},
			args:    args{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ConfigService{
				eventSyncConfig: tt.fields.eventSyncConfig,
			}
			got, _ := c.checkConfigEndpoints(tt.args.logKO, tt.args.logOK)
			if (got != "") != tt.wantErr {
				t.Errorf("checkConfigEndpoints() got = %v, wantErr %v", got, tt.wantErr)
			}
		})
	}
}

func TestConfigService_checkConfigRootValues(t *testing.T) {
	type fields struct {
		eventSyncConfig *models.EventSyncConfig
	}
	type args struct {
		logKO string
		logOK string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "with error no serviceName",
			fields: fields{
				eventSyncConfig: func() *models.EventSyncConfig {
					e := generateValidConfig()
					e.ServiceName = ""
					return e
				}(),
			},
			args:    args{},
			wantErr: true,
		},
		{
			name: "ok",
			fields: fields{
				eventSyncConfig: generateValidConfig(),
			},
			args:    args{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ConfigService{
				eventSyncConfig: tt.fields.eventSyncConfig,
			}
			got, _ := c.checkConfigRootValues(tt.args.logKO, tt.args.logOK)
			if (got != "") != tt.wantErr {
				t.Errorf("checkConfigRootValues() got = %v, wantErr %v", got, tt.wantErr)
			}
		})
	}
}

func TestConfigService_checkConfigTargetPubSub(t *testing.T) {
	type fields struct {
		eventSyncConfig *models.EventSyncConfig
	}
	type args struct {
		logKO string
		logOK string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "with error nil targetPubSub",
			fields: fields{
				eventSyncConfig: func() *models.EventSyncConfig {
					e := generateValidConfig()
					e.TargetPubSub = nil
					return e
				}(),
			},
			args:    args{},
			wantErr: true,
		},
		{
			name: "with error incorrect Topic",
			fields: fields{
				eventSyncConfig: func() *models.EventSyncConfig {
					e := generateValidConfig()
					e.TargetPubSub.Topic = "wrong/topic/defined"
					return e
				}(),
			},
			args:    args{},
			wantErr: true,
		},
		{
			name: "ok",
			fields: fields{
				eventSyncConfig: generateValidConfig(),
			},
			args:    args{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ConfigService{
				eventSyncConfig: tt.fields.eventSyncConfig,
			}
			got, _ := c.checkConfigTargetPubSub(tt.args.logKO, tt.args.logOK)
			if (got != "") != tt.wantErr {
				t.Errorf("checkConfigTargetPubSub() got = %v,wantErr %v", got, tt.wantErr)
			}
		})
	}
}

func TestConfigService_checkConfigTrigger(t *testing.T) {
	type fields struct {
		eventSyncConfig *models.EventSyncConfig
	}
	type args struct {
		logKO string
		logOK string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "with error nil trigger",
			fields: fields{
				eventSyncConfig: func() *models.EventSyncConfig {
					e := generateValidConfig()
					e.Trigger = nil
					return e
				}(),
			},
			args:    args{},
			wantErr: true,
		},
		{
			name: "with error observation period 0",
			fields: fields{
				eventSyncConfig: func() *models.EventSyncConfig {
					e := generateValidConfig()
					e.Trigger.ObservationPeriod = 0
					return e
				}(),
			},
			args:    args{},
			wantErr: true,
		},
		{
			name: "with error invalid type",
			fields: fields{
				eventSyncConfig: func() *models.EventSyncConfig {
					e := generateValidConfig()
					e.Trigger.Type = "unknown"
					return e
				}(),
			},
			args:    args{},
			wantErr: true,
		},
		{
			name: "ok type window",
			fields: fields{
				eventSyncConfig: generateValidConfig(),
			},
			args:    args{},
			wantErr: false,
		},
		{
			name: "ok type none",
			fields: fields{
				eventSyncConfig: func() *models.EventSyncConfig {
					e := generateValidConfig()
					e.Trigger.Type = models.TriggerTypeNone
					return e
				}(),
			},
			args:    args{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ConfigService{
				eventSyncConfig: tt.fields.eventSyncConfig,
			}
			got, _ := c.checkConfigTrigger(tt.args.logKO, tt.args.logOK)
			if (got != "") != tt.wantErr {
				t.Errorf("checkConfigTrigger() got = %v, wantErr %v", got, tt.wantErr)
			}
		})
	}
}
