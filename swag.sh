#!/usr/bin/env bash
set -eu -o pipefail # -x
_wd=$(pwd); _path=$(dirname $0 | xargs -i readlink -f {})

command -v swag || go install github.com/swaggo/swag/cmd/swag@latest
#### Links
# - https://swagger.io/
# - https://github.com/swaggo
# - https://github.com/swaggo/swag?tab=readme-ov-file#how-to-use-it-with-gin
# - https://github.com/swaggo/http-swagger

if [[ "${_wd}" == "${_path}" ]]; then
    echo "==> cd to target dir: ../"
    cd ../
fi

# swag_dir=swagger/docs
swag_dir=${_path}/docs
echo "==> swag dir: $swag_dir"

swag init --output $swag_dir
swag fmt --dir $swag_dir # --exclude ./internal

echo "<== swag done"
