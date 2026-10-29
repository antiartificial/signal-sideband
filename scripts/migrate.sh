#!/bin/sh
set -eu
exec sops exec-env secrets.enc.yaml 'sh scripts/apply-migrations.sh'
