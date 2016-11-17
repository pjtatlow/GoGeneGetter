# GoGeneGetter

Download test file [here](https://www.ncbi.nlm.nih.gov/geo/query/acc.cgi?acc=GSE70138). 

It's the one labeled "GSE70138_Broad_LINCS_Level2_GEX_n78980x978_2015-06-30.gct.gz". It's 144.4 Mb.

You'll need to convert it to a csv before giving it to loadBolt.

`python gct2csv.py input.gct output.csv`

Then you can use output.csv to loadBolt.
