FROM public.ecr.aws/docker/library/alpine
RUN apk update && apk add jq msitools
WORKDIR /workspace