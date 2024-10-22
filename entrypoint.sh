#!/bin/bash

set -x

MODE="${1:-prod}"

./gs-backend "${MODE}"
