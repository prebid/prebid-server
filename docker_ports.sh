#!/bin/bash
if [[ -v PORT0 ]]; then
  export PBS_PORT=${PORT0}
fi

if [[ -v PORT1 ]]; then
  export PBS_ADMIN_PORT=${PORT1}
fi