// +build go1.7

package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type testDeployer struct {
	C chan struct{}
	P chan *Payload
}

func (t testDeployer) Deploy(p *Payload) error {
	t.P <- p
	t.C <- struct{}{}
	return nil
}

type testNotifier struct {
	C chan struct{}
}

func (t testNotifier) Notify(p *Payload) {
	t.C <- struct{}{}
}

func StartRequest(payload_data string) (*http.Request, Deploy) {
	//
	token = "token"
	if payload_data == "" {
		payload_data = "{\"image\": \"company/app:latest\"}"
	}

	deployer := Deploy{
		Deployer: testDeployer{
			C: make(chan struct{}, 1),
			P: make(chan *Payload, 1),
		},
		Notifier: testNotifier{
			C: make(chan struct{}, 1),
		},
	}

	ts := httptest.NewServer(deployer.getRoutes())
	defer ts.Close()

	url := ts.URL + "/service/test"
	req, _ := http.NewRequest("PUT", url, strings.NewReader(payload_data))
	req.Header.Set("Authorization", token)

	client := http.Client{}
	_, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	return req, deployer
}

func TestAuthorizer(t *testing.T) {

	token = "TEST_TOKEN"

	td := []struct {
		name     string
		value    string
		expected interface{}
	}{
		{"Missing token", "", http.StatusUnauthorized},
		{"Wrong token", "whatever", http.StatusForbidden},
		{"Correct token", token, http.StatusOK},
	}

	handler := Authorize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("HI"))
	}))

	for _, tc := range td {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			req := httptest.NewRequest("PUT", "/somewhere", nil)
			req = req.WithContext(context.Background())
			req.Header.Set("Authorization", tc.value)

			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			if recorder.Code != tc.expected {
				t.Errorf("Expecting http code %v got %v", tc.expected, recorder.Code)
			}
		})
	}
}

func TestDeployEndpointMethods(t *testing.T) {
	t.Skip()
}

func TestDeployEndpointParsesPayload(t *testing.T) {

	_, deployer := StartRequest("")
	image_name := "company/app:latest"

	p := deployer.Deployer.(testDeployer).P

	for {
		select {
		case payload := <-p:
			if payload.Artifact != image_name {
				t.Errorf("Expecting %s, got %v\n", image_name, payload.Artifact)
			}
			if payload.ServiceName != "test" {
				t.Errorf("Expecting ServiceName of test, got %v", payload.ServiceName)
			}
			return
		case <-time.After(time.Second):
			t.Errorf("Expecting /service endpoint to call Deploy().  Timed out.\n")
			return
		}
	}

}

func TestDeployServeHTTPCodes(t *testing.T) {

	td := []struct {
		name     string
		deploy   Deploy
		req      *http.Request
		expected int
	}{
		{"TestDeployRouteChecksAuthorization", Deploy{}, httptest.NewRequest("PUT", "/s", nil), http.StatusForbidden},
		{"TestDeployEndPointMalformedJSON", Deploy{}, func() *http.Request {
			r := httptest.NewRequest("PUT", "/s", strings.NewReader("{image"))
			return r.WithContext(context.WithValue(context.Background(), "Authorized", true))
		}(), http.StatusBadRequest},
		{"TestInvalidServiceName", Deploy{}, func() *http.Request {
			r := httptest.NewRequest("PUT", "/--abc1234..", nil)
			return r.WithContext(context.WithValue(context.Background(), "Authorized", true))
		}(), http.StatusBadRequest},
	}

	for _, tc := range td {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			rec := httptest.NewRecorder()

			tc.deploy.ServeHTTP(rec, tc.req)

			if rec.Code != tc.expected {
				t.Errorf("Expecting HTTP code %d, got %d", tc.expected, rec.Code)
			}
		})
	}
}

func TestValidService(t *testing.T) {
	deployer := Deploy{}

	if !deployer.Service("test") {
		t.Errorf("Expecting true, got false")
	}
}

func TestEndpointCallsDeploy(t *testing.T) {

	_, deployer := StartRequest("")
	called := deployer.Deployer.(testDeployer).C

	for {
		select {
		case <-called:
			return
		case <-time.After(time.Second):
			t.Errorf("Expecting /service endpoint to call Deploy().  Timed out.\n")
			return
		}
	}
}

func TestEndpintNotifys(t *testing.T) {

	_, deployer := StartRequest("")
	called := deployer.Deployer.(testDeployer).C

	for {
		select {
		case <-called:
			return
		case <-time.After(time.Second):
			t.Errorf("Expecting /service endpoint to call Notify().  Timed out.\n")
			return
		}
	}
}
