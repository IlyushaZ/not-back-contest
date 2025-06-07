#!/bin/bash

VEGETA_DURATION=${VEGETA_DURATION:-20s}
VEGETA_RPS=${VEGETA_RPS:-600}

# Start docker containers in detached mode
docker compose --profile perftest up -d --force-recreate # run everything except crond with items-generator

sleep 10 # make sure all services have started (don't have time to make something better)

# Run the target generator script
$(pwd)/test/generate_targets.sh

# Run test via vegeta and output results to stdout and results.bin so it can be analyzed in different ways in the future.
vegeta attack -duration=${VEGETA_DURATION} --rate=${VEGETA_RPS}/1s -targets=test/targets.txt | tee results.bin | vegeta report

docker compose down
