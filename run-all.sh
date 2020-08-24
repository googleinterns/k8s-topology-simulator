#!/bin/bash
# Copyright 2020 Google LLC
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     https://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

declare -a algs=("SharedGlobal" "SharedMultiZone" "Local" "LocalWeighted" "LocalOpt" "LocalShared" "Original")
if [ $# -gt 0 ]
then file=$1
else file=./data/range-input.csv
fi

if [ ! -f $file ]
then echo "input file $file doesn't exist, please run ./data/range-input-generator.py first"; exit
fi

for alg in ${algs[@]}; do
    echo "Running $alg"
    go run main.go -input=$file -alg=$alg -output=./data/$alg-range-output.csv
done
