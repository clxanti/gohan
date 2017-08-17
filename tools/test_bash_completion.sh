#!/bin/bash
set -e

(
  cd tools

  source ./bash_completion.sh
  ./bash_completion_tests.py

  source ./gohan_client_bash_completion.sh
  ./gohan_client_bash_completion_tests.py
)

