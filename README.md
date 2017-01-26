# GoGeneGetter

Download test file [here](https://www.ncbi.nlm.nih.gov/geo/query/acc.cgi?acc=GSE70138). 

It's the one labeled "GSE70138_Broad_LINCS_Level2_GEX_n78980x978_2015-06-30.gct.gz". It's 144.4 Mb.

You'll need to convert it to a csv before giving it to loadBolt.

`python gct2csv.py input.gct output.csv`

Then you can use output.csv to loadBolt.

## To load the database from the `loadBolt/` directory:

```
go run loadBolt.go --file ../data/small_file.csv --db ../test.db --meta [1,2,3,4,5,6,7,8,9,10,11]
```

- `file`: the location of the input file
- `db`: the location where you want the database stored
- `meta`: the meta-columns; 0-indexed
- `id`(optional): the column containing the id; defaults to 0

## To run the server:

Make sure your `config.json` file has the correct path to the database.

From the `server/` directory:

```
go run testIris.go config.json
```
