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

# this script is used to calculate aggregate results for different algorithms,
# python3 required
import sys
import csv
import os

# python3 is requried
assert sys.version_info[0] >= 3

from utils import print_progress_bar

class metric:
    def __init__(self):
        self.value = 0.
        self.max_value = float("-inf")
        self.min_value = float("inf")
        self.max_cases = list()
        self.min_cases = list()
    def process(self, name, value):
        self.value += value
        if value > self.max_value:
            self.max_value = value
            self.max_cases.clear()
        if value == self.max_value:
            self.max_cases.append(name)

        if value < self.min_value:
            self.min_value = value
            self.min_cases.clear()
        if value == self.min_value:
            self.min_cases.append(name)

# helper function helps calculate aggregate results of an algorithm
def process_one_file(file_path, writer, alg_name):
    file_path = file_path.strip()
    file_path = os.path.normpath(os.path.join(os.path.dirname(os.path.realpath(__file__)), "./{0}".format(file_path)))
    try:
        with open(file_path, mode='r') as csv_file:
            valid_records = 0
            invalid_records = 0
            # score aggregate results
            total_score = metric()
            inzone_score = metric()
            deviation_score = metric()
            slice_score = metric()
            max_deviation = metric()
            csv_reader = csv.DictReader(csv_file)
            for row in csv_reader:
                if row['score'] == 'invalid':
                    invalid_records += 1
                    continue
                valid_records += 1
                total_score.process(row['input name'], float(row['score']))
                inzone_score.process(row['input name'], float(row['in-zone-traffic score']))
                deviation_score.process(row['input name'], float(row['deviation score']))
                slice_score.process(row['input name'], float(row['slice score']))
                max_deviation.process(row['input name'], float(row['max deviation'].strip('%')))
            row_data = dict()
            row_data['alg name'] = alg_name
            row_data['invalid cases'] = invalid_records
            row_data['valid cases'] = valid_records
            row_data['mean total score'] = "%.2f" % float(total_score.value/valid_records)
            row_data['max total score'] = "%.2f" % float(total_score.max_value)
            row_data['min total score'] = "%.2f" % float(total_score.min_value)
            row_data['mean inzone score'] = "%.2f" % float(inzone_score.value/valid_records)
            row_data['max inzone score'] = "%.2f" % float(inzone_score.max_value)
            row_data['min inzone score'] = "%.2f" % float(inzone_score.min_value)
            row_data['mean deviation score'] = "%.2f" % float(deviation_score.value/valid_records)
            row_data['max deviation score'] = "%.2f" % float(deviation_score.max_value)
            row_data['min deviation score'] = "%.2f" % float(deviation_score.min_value)
            row_data['mean slice score'] = "%.2f" % float(slice_score.value/valid_records)
            row_data['max slice score'] = "%.2f" % float(slice_score.max_value)
            row_data['min slice score'] = "%.2f" % float(slice_score.min_value)
            row_data['max deviation %'] = "%.2f" % float(max_deviation.max_value) + '%'
            writer.writerow(row_data)
    except Exception as err:
        print(err, "move to the next file")

# ask user to specify file names for different algorithms
def user_input_files(alg_output_files, alg_names):
    print("current output files for {0} are :\n{1}".format(alg_names, alg_output_files))
    change = input("change files? y/n\n")
    if change == 'y':
        change = True
    elif change == 'n':
        change = False
    else:
        print("unexpected input\n")
        return user_input_files(alg_output_files, alg_names)

    if change == False:
        return alg_output_files
    
    new_files = input("input {0} files names (use comma to seperate each):\n".format(len(alg_names)))
    new_files = new_files.split(',')
    if len(new_files) != len(alg_names) :
        print("unmatched number of files for {0} algorithms\n".format(len(alg_names)))
        return user_input_files(alg_output_files, alg_names)
    return new_files

def main():
    alg_names = ['original', 'local', 'local-shared', 'local-weighted', 'local-opt', 'shared-global', 'shared-multizone']
    # default file names
    alg_output_files = ['original-alg-output.csv', 'local-alg-output.csv', 'local-shared-alg-output.csv', 'local-weighted-alg-output.csv', 'local-opt-alg-output.csv', 'shared-global-alg-output.csv', 'shared-multizone-alg-output.csv']
    # interactively ask user for file names
    alg_output_files = user_input_files(alg_output_files, alg_names)
    print("---"*10)
    print("processing files : {0}".format(alg_output_files))

    output_path = os.path.normpath(os.path.join(os.path.dirname(os.path.realpath(__file__)), "./results.csv"))
    try:
        with open(output_path, mode='w') as csv_output:
            field_names = ['alg name', 'invalid cases', 'valid cases', 'mean total score', 'max total score', 'min total score', 'mean inzone score', 'max inzone score', 'min inzone score', 'mean deviation score', 'max deviation score', 'min deviation score', 'mean slice score', 'max slice score', 'min slice score', 'max deviation %']
            writer = csv.DictWriter(csv_output, fieldnames=field_names)
            writer.writeheader()
            for index, alg_name in enumerate(alg_names):
                process_one_file(alg_output_files[index], writer, alg_name)
                print_progress_bar(index+1, len(alg_names), length=10)
    except Exception as err:
        print(err)
        sys.exit()

if __name__ == "__main__":
    main()
