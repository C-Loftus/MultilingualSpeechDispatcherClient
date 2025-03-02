#!/bin/sh

set -eu

# change directory to the project root so that we can run go run 
# and have relative paths work
cd "$(dirname "$0")/.."

cat ./example/bilingual_conversation.txt | go run . --use-languages Spanish,English