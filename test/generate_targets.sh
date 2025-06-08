#!/usr/bin/env bash

: > test/targets.txt

declare -A used
used_list=() # list to take repeating item_id's
min_id=1
max_id=11000

uniques_count=0
total_requests=10000

for ((i=1; i<=total_requests; i++)); do
    # 10% chance to take repeating id from generated ones
    if (( RANDOM % 3 == 0 )) && (( uniques_count > 0 )); then
        idx=$(( RANDOM % uniques_count ))
        item_id=${used_list[idx]}
    else
    # generate unique item_id in range min_id..max_id
    range=$(( max_id - min_id + 1 ))
    while :; do
        candidate=$(( RANDOM % range + min_id ))
        [[ -z "${used[$candidate]}" ]] && break
    done
    item_id=$candidate
    used[$item_id]=1
    used_list+=("$item_id")
    uniques_count=$(( uniques_count + 1 ))
    fi

  # random user_id in range 1..100
  user_id=$(( RANDOM % 100 + 1 ))

  printf 'POST http://localhost:8000/checkout?user_id=%d&item_id=%d\n' "$user_id" "$item_id" >> test/targets.txt
done
