FROM alpine:3.21.2

RUN apk update && apk upgrade && \
  apk --no-cache add bash openssh rsync cifs-utils nano && \
  ssh-keygen -A && sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config

# Setup SSHsudo apk add
# https://docs.docker.com/engine/examples/running_ssh_service/
EXPOSE 22

COPY ./resources /

CMD ["/startup.sh"]