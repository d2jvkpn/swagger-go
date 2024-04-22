#!/usr/bin/env bash
set -eu -o pipefail # -x
_wd=$(pwd); _path=$(dirname $0 | xargs -i readlink -f {})

####
command -v swag || go install github.com/swaggo/swag/cmd/swag@latest

# swag_dir=swagger-go/docs
swag_dir=${_path}/docs
echo "==> swag dir: $swag_dir"

swag init --output $swag_dir
swag fmt --dir $swag_dir # --exclude ./internal

echo "<== swag done"

####
cd ${_path}

# build_time=$(date +'%FT%T.%N%:z')
build_time=$(date +'%FT%T%:z')
git_repository=$(git config --get remote.origin.url)
git_branch=$(git rev-parse --abbrev-ref HEAD)
git_commit_id=$(git rev-parse --verify HEAD)
git_commit_time=$(git log -1 --format="%at" | xargs -I{} date -d @{} +%FT%T%:z)

git_tree_state="clean"
uncommitted=$(git status --short)
unpushed=$(git diff origin/$git_branch..HEAD --name-status)
# [ ! -z "$uncommitted$unpushed" ] && git_tree_state="dirty"
[ ! -z "$unpushed" ] && git_tree_state="unpushed"
[ ! -z "$uncommitted" ] && git_tree_state="uncommitted"


mkdir -p target

GO_ldflags="-X main.build_time=$build_time \
  -X main.git_repository=$git_repository \
  -X main.git_branch=$git_branch \
  -X main.git_commit_id=$git_commit_id \
  -X main.git_commit_time=$git_commit_time \
  -X main.git_tree_state=$git_tree_state"

echo "==> $GO_ldflags"

set -x

go build -ldflags="-w -s $GO_ldflags" -o target/swagger-go main.go
