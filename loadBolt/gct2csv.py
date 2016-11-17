#!/usr/bin/env python

import sys, gzip

in_file = ""


if sys.argv[1][-2:] == "gz":
    in_file = gzip.open(sys.argv[1],'r')
else:
    in_file = open(sys.argv[1],'r')

line1 = in_file.readline()
line2 = in_file.readline()

cols_to_ignore = []

header = in_file.readline()
header = header.strip().split('\t')
old_header = []
new_rows = {}
for i in xrange(1,len(header)):
    item = header[i]
    if item[0:2] == "pr":
        cols_to_ignore.append(i)
    else:
        new_rows[i] = [item]

new_cols = []

new_cols.append(header[0])

for line in in_file:
    line = line.strip().split('\t')
    new_cols.append(line[0])
    for i in xrange(1,len(line)):
        if i not in cols_to_ignore:
            item = line[i]
            item = item.replace(",","_")
            new_rows[i].append(item)

in_file.close()

print "Finished reading"

with open(sys.argv[2],'w') as out_file:
    out_file.write(','.join(new_cols))
    out_file.write('\n')
    for key in new_rows:
        out_file.write(','.join(new_rows[key]))
        out_file.write('\n')

print "Done"

