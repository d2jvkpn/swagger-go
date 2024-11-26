#!/bin/bash
set -eu -o pipefail # -x
_wd=$(pwd); _path=$(dirname $0 | xargs -i readlink -f {})

export IMAGE_Tag=$1 HTTP_Port=$2

export IMAGE_Name=$(yq .image_name project.yaml) \
  APP_Name=$(yq .app_name project.yaml) \
  USER_UID=$(id -u) \
  USER_GID=$(id -g)

envsubst < ${_path}/compose.template.yaml > compose.yaml

exit 0

# docker-compose pull
[ ! -z "$(docker ps --all --quiet --filter name=$container)" ] &&
  docker rm -f $container
# 'docker-compose down' removes running containers only, not stopped containers

# USER_UID=$USER_UID USER_GID=$USER_GID docker-compose up -d
docker-compose up -d

exit 0

docker stop $container && docker stop $container
docker rm -f $container
