#!/bin/bash
set -eu -o pipefail

pkg-config --modversion gtk+-3.0 |
sed 's/\.[^.]*$//g' |
sed 's/\./_/g' |
sed 's/^/gtk_/g'
