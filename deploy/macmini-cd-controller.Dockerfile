FROM docker:cli

RUN apk add --no-cache bash ca-certificates git
COPY macmini-cd-controller.sh /usr/local/bin/macmini-cd-controller.sh
RUN chmod 0755 /usr/local/bin/macmini-cd-controller.sh

ENTRYPOINT ["/usr/local/bin/macmini-cd-controller.sh"]
