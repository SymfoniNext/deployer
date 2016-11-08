FROM alpine:3.4
ADD deployer /bin/deployer
ENV DOCKER_HOST=unix:///var/run/docker.sock
ENV BIND=:9999
ENTRYPOINT ["/bin/deployer"]
EXPOSE 9999
