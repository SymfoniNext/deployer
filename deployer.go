// +build go1.7

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/go-zoo/bone"
	"github.com/namsral/flag"
)

var (
	docker_host  string
	listen_addr  string
	notify_flags string
	token        string
)

type Notifier interface {
	Notify(*Payload)
}

type Deployer interface {
	Deploy(*Payload) error
}

type Payload struct {
	ServiceName string
	Artifact    string `json:"image"`
}

type Deploy struct {
	routes http.Handler // cache

	Deployer Deployer
	Notifier Notifier
}

// This isn't right ...
func (d Deploy) getRoutes() http.Handler {
	if d.routes == nil {
		mux := bone.New()
		mux.Put("/service/#name^[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]$", Authorize(d))
		d.routes = mux
	}
	return d.routes
}

// Return whether or not the named service is a valid service for deploying
func (d Deploy) Service(s string) bool {
	// Service route filters the name via regexp, so just need to check we have a value
	return s != ""
}

func (d Deploy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Context().Value("Authorized") != true {
		log.Printf("No authorized context set\n")
		http.Error(w, "You are not authorized to perform this request", http.StatusForbidden)
		return
	}

	// @todo? 405 for wrong method if not put

	service_name := bone.GetValue(r, "name")
	if !d.Service(service_name) {
		log.Printf("Invalid service name supplied\n")
		http.Error(w, "Unknown service name", http.StatusBadRequest)
		return
	}

	p := &Payload{
		ServiceName: service_name,
	}

	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		log.Println("Error decoding body:", err)
		http.Error(w, "Malformed request", http.StatusBadRequest)
		return
	}

	log.Printf("Running deploy with payload %s\n", p)
	if err := d.Deployer.Deploy(p); err != nil {
		log.Printf("Error deploying %s: %s\n", p.ServiceName, err)
		http.Error(w, "Unable to deploy", http.StatusInternalServerError)
		return
	}

	d.Notifier.Notify(p)
}

func init() {
	flag.StringVar(&docker_host, "docker-host", "unix:///var/run/docker.sock", "Address of Docker host")
	flag.StringVar(&listen_addr, "bind", ":9999", "Listen address and port")
	flag.StringVar(&token, "token", "", "Use this token for authentication of requests")
	flag.StringVar(&notify_flags, "notify-flags", "", "Flags to pass to notify (as JSON)")
}

func main() {
	flag.Parse()

	if token == "" {
		help()
		os.Exit(1)
	}

	if notify_flags == "" {
		help()
		os.Exit(1)
	}

	var nflags map[string]string
	nflagData := bytes.NewBufferString(notify_flags).Bytes()
	if err := json.Unmarshal(nflagData, &nflags); err != nil {
		log.Fatal("Error decoding notify_flags\n")
	}

	nAddr := nflags["addr"]
	nTube := nflags["tube"]
	nBody, err := base64.StdEncoding.DecodeString(nflags["template"])
	if err != nil {
		log.Fatal(err)
	}

	docker_client, err := docker.NewClient(docker_host)
	if err != nil {
		log.Fatal(err)
	}

	deployer := Deploy{
		Deployer: DockerServiceDeploy{
			client: docker_client,
		},
		Notifier: NewNotifyBeanstalkd(nAddr, nTube, string(nBody)),
	}

	server := &http.Server{
		Addr:         listen_addr,
		Handler:      deployer.getRoutes(),
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 10,
	}
	log.Println(server.ListenAndServe())
}

func Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//
		userToken := r.Header.Get("Authorization")
		if userToken == "" {
			// Not quite legal..
			log.Printf("No Authorization header supplied\n")
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}
		if userToken != token {
			log.Printf("Invalid token provided\n")
			http.Error(w, "Invalid token provided", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), "Authorized", true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func help() {
	flag.Usage()
}
