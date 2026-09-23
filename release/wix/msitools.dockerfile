FROM alpine
RUN apk update && apk add jq msitools
WORKDIR /workspace