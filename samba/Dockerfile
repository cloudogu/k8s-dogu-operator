FROM dockurr/samba

RUN apk update && apk upgrade && \
  apk --no-cache add nano

RUN echo "root:root" | chpasswd

COPY resources /

CMD ["/startup.sh"]