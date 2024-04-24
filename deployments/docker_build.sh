#!/bin/bash
set -eu -o pipefail # -x
_wd=$(pwd); _path=$(dirname $0 | xargs -i readlink -f {})

#### load
[ $# -eq 0 ] && { >&2 echo "Argument {branch} is required!"; exit 1; }

git_branch=$1

app_name=$(yq .app project.yaml)
app_version=$(yq .version project.yaml)
image_name=$(yq .image_name project.yaml)

tag=${git_branch}-$(yq .version project.yaml)
tag=${DOCKER_Tag:-$tag}
image=$image_name:$tag
build_time=$(date +'%FT%T.%N%:z')

# env variables
GIT_Pull=$(printenv GIT_Pull || true)
DOCKER_Pull=${DOCKER_Pull:-"true"}
DOCKER_Push=${DOCKER_Push:-"true"}
BUILD_Region=${BUILD_Region:-""}

[ -f .env ] && {
  2>&1 echo "==> load .env"
  . .env
}

#### git
function on_exit() {
    if [ ! -z $(git branch -a | awk '$1=="dev"{print 1; exit}') ]; then
        git checkout dev # --force
    fi
}
trap on_exit EXIT

git checkout $git_branch

[[ "$GIT_Pull" != "false" ]] && git pull --no-edit

git_repository="$(git config --get remote.origin.url)"
git_branch="$(git rev-parse --abbrev-ref HEAD)" # current branch
git_commit_id=$(git rev-parse --verify HEAD) # git log --pretty=format:'%h' -n 1
git_commit_time=$(git log -1 --format="%at" | xargs -I{} date -d @{} +%FT%T%:z)

git_tree_state="clean"
uncommitted=$(git status --short)
unpushed=$(git diff origin/$git_branch..HEAD --name-status)
# [[ ! -z "$uncommitted$unpushed" ]] && git_tree_state="dirty"
[[ ! -z "$unpushed" ]] && git_tree_state="unpushed"
[[ ! -z "$uncommitted" ]] && git_tree_state="uncommitted"

####
echo "==> docker build $image"

[[ "$DOCKER_Pull" != "false" ]] && \
for base in $(awk '/^FROM/{print $2}' ${_path}/Dockerfile); do
    echo ">>> pull $base"
    docker pull $base
    bn=$(echo $base | awk -F ":" '{print $1}')
    if [[ -z "$bn" ]]; then continue; fi
    docker images --filter "dangling=true" --quiet "$bn" | xargs -i docker rmi {}
done

echo ">>> build image: $image..."

GO_ldflags="-X main.build_time=$build_time \
  -X main.git_branch=$git_branch \
  -X main.git_commit_id=$git_commit_id \
  -X main.git_commit_time=$git_commit_time \
  -X main.git_tree_state=$git_tree_state \
  -X main.image=$image"

docker build --no-cache --file ${_path}/Dockerfile \
  --build-arg=BUILD_Region="$BUILD_Region" \
  --build-arg=APP_Name="$app_name" \
  --build-arg=APP_Version="$app_version" \
  --build-arg=GO_ldflags="$GO_ldflags" \
  --tag $image ./

docker image prune --force --filter label=stage=${app_name}_builder &> /dev/null

#### push image
[ "$DOCKER_Push" != "false" ] && docker push $image

docker images --filter "dangling=true" --quiet $image_name | xargs -i docker rmi {}
