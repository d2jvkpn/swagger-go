#!/usr/bin/env bash
set -eu -o pipefail # -x
_wd=$(pwd); _path=$(dirname $0 | xargs -i readlink -f {})

command -v swag || go install github.com/swaggo/swag/cmd/swag@latest

if [ $# -gt 0 ]; then
    target_dir=$1
    echo "==> cd to target dir: $target_dir"
    cd $target_dir
fi

# swag_dir=swagger-go/docs
swag_dir=${_path}/docs
echo "==> swag dir: $swag_dir"

swag init --output $swag_dir
swag fmt --dir $swag_dir # --exclude ./internal

echo "<== swag done"
