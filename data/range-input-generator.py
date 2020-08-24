#! /usr/bin/env python3
"""
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
"""

# this script is used to generate large input dataset for simulator, python3
# required.
import sys
import csv
import os
from itertools import combinations_with_replacement as combinations

# python3 is requried
assert sys.version_info[0] >= 3

from utils import print_progress_bar


# helper function helps generate one category of range inputs, default nodes =
# 30 and endpoints range from 0-100 for three zones
def generate_one_section(writer,nodes=[30, 30, 30], ep_range=[0,101], step=1, suffix='', bar=True):
    name = 0
    section_name = "{0}-{1} endpoints {2} nodes, step: {3}".format(ep_range[0], ep_range[1]-1, nodes, step)
    endpoints = list(combinations(range(ep_range[0], ep_range[1], step), len(nodes)))
    total = len(endpoints)
    for comb in endpoints:
        row_data = dict()
        row_data[field_names[0]] = str(name) + suffix
        for index, ep in enumerate(comb):
            row_data[field_names[index+1]] = "{node} {endpoint}".format(node=nodes[index], endpoint=ep)
        writer.writerow(row_data)
        name += 1
        if bar:
            print_progress_bar(name, total, progress=section_name, length = 70)

file_dir = os.path.normpath(os.path.join(os.path.dirname(os.path.realpath(__file__)), "./range-input.csv"))
print("creating range input to :" + file_dir)
with open(file_dir, mode='w') as csv_file:
    field_names = ['name', 'zone1', 'zone2', 'zone3']
    writer = csv.DictWriter(csv_file, fieldnames=field_names)

    writer.writeheader()
    # generate range input: 3 zones
    # nodes are the conbinations from 1-10
    # endpoints range from 0-100 with step = 1
    nodes_comb = list(combinations(range(1, 11), 3))
    for index, nodes in enumerate(nodes_comb):
        generate_one_section(writer, nodes, suffix='-'+str(nodes), bar=False)
        print_progress_bar(index+1, len(nodes_comb), progress="0-100 endpoints 1-10 nodes, step: 1")

    # generate range input: 3 zones
    # nodes = 30 for every zone
    # endpoints range from 100-1000 with step = 7
    generate_one_section(writer, ep_range=[100, 1001], step = 7, suffix='-high')
