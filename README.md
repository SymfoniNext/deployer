Deployer
--------

A REST API for updating Docker services with a new image.  *Running `docker service update` the fancy way.*

Add this server to the end of your CD workflow, and maybe life will be *a little easier*.   A token (via HTTP headers) is used for authorisation and notifications are pushed into beanstalkd; so you can write any type of notifier you want. 

# Running deployer 

There is an external requirement on `beanstalkd` for sending notifications (it probably works without it, but we haven't tried).  beanstalkd is pretty easy to setup.

All requests to the server should be done over SSL; we use a SSL termination proxy and no doubt you have something similar setup.

## Server flags/options

`TOKEN`

This string must be provided via the **Authorization** HTTP header for any update request to stand a chance of succeeding.

`DOCKER_HOST`

Address of the Docker daemon - this must be a Swarm manager node.

`NOTIFY_FLAGS`

This one is a bit more tricky; flags that will be passed to the notification system.  Currently only beanstalkd is supported and the string must be in JSON format.

```
{
  "addr": "addr_to_beanstalkd:11300",
  "tube": "name_of_tube_to_put_messages_into",
  "template": "base64 encoded message to put into beanstalk, available variables are {{.Artifact}} and {{.ServiceName}}"
}
```

So to push the message `I just deployed image/name:tag to service_name` you would write the template `"I just deployed {{.Artifact}} for {{.ServiceName}}` and base64 encode it.

```
{
  "addr": "beanstalk:11300",
  "tube": "jobs",
  "template": "IkkganVzdCBkZXBsb3llZCB7ey5BcnRpZmFjdH19IGZvciB7ey5TZXJ2aWNlTmFtZX19"
}
```

`BIND`

Local address for the server to bind to.

## Starting the service

We will run deployer as a Docker service, ensuring it has access to the host node's Docker daemon and only on Swarm manager nodes. 

```
$ docker service create --name deployer --network your_overlay_network --mount type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock --constraint 'node.role == manager' --env NOTIFY_FLAGS='{"addr": "beanstalk:11300","tube": "jobs","template": "IkkganVzdCBkZXBsb3llZCB7ey5BcnRpZmFjdH19IGZvciB7ey5TZXJ2aWNlTmFtZX19"}' --env TOKEN=SPECIFIY_SOME_TOKEN_HERE symfoni/deployer:latest
```

## Point your proxy to the server

It is assumed you know how to do this. 


## Marking services as being updatable

Add the following label to the service: `deployer.allowUpdates=true`.

```
$ docker service update --label-add 'deployer.allowUpdates=true' service_name
```

# Notifications

As mentioned already, notifications are sent via beanstalkd (see `NOTIFY_FLAGS` on how to configure).  You will need a consumer to process these messages, and that consumer canthen do anything it wants with it (send to Slack, email, IRC etc.).  A notification will be sent if we get a successful response back from the Docker API, this doesn't mean that the actual deployment was a success.


# Using the server

Only `PUT` requests to the `/service/:service-name` endpoint will do anything.  You need to include a JSON payload with the request in the following format:

```
{
  "image": "nameofimage:tag"
}
```

That will update `:service-name` to use the image `nameofimage:tag`.

## curl example 

This will update our service `blog` with image `me/blog:v1.12`.  This command is run by our CD server after a successfull image hub push.

```
$ curl -v --header "Authorization: SPECIFIY_SOME_TOKEN_HERE" -X PUT -d '{"image": "me/blog:v1.12"}' https://your.deployment.url/service/blog
```

# Compiling a binary

You need Go > v1.7, we typed the following and got a nice binary:

```
$ CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -extld ld -extldflags -static' -a -x -o deployer .
```
