package main

import (
	"bytes"
	"testing"
)

func TestNotifyBeanstalkTemplatePayload(t *testing.T) {

	body_template := "{\"msg\": \"Deployed {{.ServiceName}} as {{.Artifact}}\"}"
	expected := []byte(`{"msg": "Deployed test as company/app:latest"}`)

	notify := NewNotifyBeanstalkd("", "", body_template)

	payload := &Payload{
		ServiceName: "test",
		Artifact:    "company/app:latest",
	}

	body := notify.parseBody(payload)

	if !bytes.Equal(body, expected) {
		t.Errorf("Expecting '%s', got '%v'", expected, body)
	}
}

func TestNotifyBeanstalkd(t *testing.T) {
	t.Skip()

	addr := "172.17.0.9:11300"
	tube := "jobs"
	body_template := "{\"Image\": \"symfoni/slack-post:latest\", \"Cmd\": [\"172.19.0.8:8500\", \"deployer\", \"hello I deployed {{.ServiceName}} ({{.Artifact}})\"]}"
	notify := NewNotifyBeanstalkd(addr, tube, body_template)

	notify.Notify(&Payload{"some-service", "company/app:12345"})
}
