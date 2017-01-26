package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"index/suffixarray"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/deckarep/golang-set"
	"github.com/kataras/iris"
)

var dbFile string
var cacheMeta bool
var cachedMetaNames map[string]int
var cachedMetaValues map[string][]string
var searchIndex *suffixarray.Index

const (
	delim = '?'
)

func main() {
	parseArgs()

	searchIndex = suffixarray.New([]byte(""))
	fmt.Println("Reading index")
	r, _ := os.Open("genes.index")
	searchIndex.Read(r)

	iris.OnError(iris.StatusNotFound, func(ctx *iris.Context) {
		ctx.ServeFile("./site/index.html", false)
	})

	// iris.Post("/query", func(ctx *iris.Context) {
	// 	ctx.Response.Header.Set("Content-Type", "text/csv")
	// 	var query map[string][]string
	// 	queryString := ctx.PostValuesAll()["query"][0]
	// 	json.Unmarshal([]byte(queryString), &query)

	// 	ctx.Response.Header.Set("Content-disposition", "attachment; filename=test.csv")
	// 	// ctx.StreamWriter(stream)

	// 	ctx.Stream(func(w *bufio.Writer) {

	// 		db, err := bolt.Open(dbFile, 0600, &bolt.Options{ReadOnly: true})
	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}
	// 		defer db.Close()
	// 		samples, properties := parseQuery(db, query)
	// 		fmt.Fprint(w, strings.Join(properties, ","))
	// 		fmt.Fprint(w, "\n")

	// 		db.View(func(tx *bolt.Tx) error {
	// 			for _, sample := range samples {

	// 				b := tx.Bucket([]byte(sample))
	// 				for _, prop := range properties {
	// 					v := b.Get([]byte(prop))
	// 					fmt.Fprint(w, string(v)+",")
	// 				}
	// 				fmt.Fprint(w, "\n")
	// 				if err := w.Flush(); err != nil {
	// 					return err
	// 				}
	// 			}
	// 			return nil
	// 		})
	// 	})

	// })

	iris.Get("/numSamples", func(ctx *iris.Context) {

		var query map[string][]string
		query_string := ctx.URLParams()["query"]
		json.Unmarshal([]byte(query_string), &query)

		db, err := bolt.Open(dbFile, 0600, &bolt.Options{ReadOnly: true})
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		samples, _ := parseQuery(db, query)
		numSamples := len(samples)
		ctx.Write(strconv.Itoa(numSamples))

	})

	iris.Get("/meta", func(ctx *iris.Context) {
		queryString := ctx.URLParams()["query"]
		var query map[string][]string
		json.Unmarshal([]byte(queryString), &query)
		fmt.Println("Got query:", query)
		values, _ := json.Marshal(getFilteredMeta(query))
		ctx.WriteString(string(values))

		// if !cacheMeta {
		// 	if len(q) == 0 { // we want all meta names from database
		// 		names, _ := json.Marshal(getMetaNames())
		// 		ctx.WriteString(string(names))
		// 	} else { // we want the values of a specific name from database
		// 		values, _ := json.Marshal(getMetaValues(q))
		// 		ctx.WriteString(string(values))
		// 	}
		// } else {
		// 	if len(q) == 0 { // we want all meta names from cache
		// 		names, _ := json.Marshal(cachedMetaNames)
		// 		ctx.WriteString(string(names))
		// 	} else { // we want the values of a specific name from cache
		// 		values, _ := json.Marshal(cachedMetaValues[q])
		// 		ctx.WriteString(string(values))
		// 	}
		// }
	})

	iris.Get("/info", func(ctx *iris.Context) {
		values := getMetaValues("genes")
		info := make(map[string]interface{})
		info["numGenes"] = len(values["genes"])
		infoString, _ := json.Marshal(info)

		ctx.WriteString(string(infoString))
	})

	iris.Get("/api/genes", func(ctx *iris.Context) {
		queryString := ctx.URLParams()["query"]
		fmt.Println("GENE QUERY:", queryString)

		genes := make([]string, 0)
		done := make(chan bool, 1)

		go search(queryString, done, &genes)

		go func() {
			time.Sleep(5 * time.Millisecond)
			done <- false
		}()

		valid := <-done
		response := make(map[string]interface{})
		response["valid"] = valid
		response["genes"] = genes
		responseString, _ := json.Marshal(response)

		ctx.WriteString(string(responseString))
		ctx.SetConnectionClose()
	})

	iris.StaticServe("./site/css", "/css")
	iris.StaticServe("./site/js", "/js")
	iris.StaticServe("./site/templates", "/templates")

	iris.Get("/", func(ctx *iris.Context) {
		ctx.ServeFile("./site/index.html", false)
	})

	iris.Listen(":8080")
}

func parseArgs() {
	f, fErr := os.Open(os.Args[1])
	if fErr != nil {
		panic(fErr)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	configJson := ""
	for scanner.Scan() {
		configJson += scanner.Text()
	}

	var config map[string]interface{}
	json.Unmarshal([]byte(configJson), &config)
	cacheMeta = config["cacheMeta"].(bool)
	dbFile = config["dbFile"].(string)

	if cacheMeta {
		cachedMetaValues = getMeta()
		cachedMetaNames = make(map[string]int)
		for name, values := range cachedMetaValues {
			cachedMetaNames[name] = len(values)
		}
	}

}

func getMetaValues(name string) map[string][]string {
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{ReadOnly: true})
	defer db.Close()
	if err != nil {
		log.Fatal(err)
		return nil
	} else {
		meta := make(map[string][]string)
		db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("meta"))

			vals := b.Get([]byte(name))
			metaVals := make([]string, 0)
			json.Unmarshal(vals, &metaVals)
			meta[name] = metaVals

			return nil
		})
		return meta
	}
}

func getMetaNames() map[string]int {
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{ReadOnly: true})
	defer db.Close()
	if err != nil {
		log.Fatal(err)
		return nil
	} else {
		meta := make(map[string]int)
		db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("meta"))
			v := b.Get([]byte("names"))
			metaNames := make([]string, 0)
			json.Unmarshal(v, &metaNames)

			for _, name := range metaNames {
				vals := b.Get([]byte(name))
				metaVals := make([]string, 0)
				json.Unmarshal(vals, &metaVals)
				meta[name] = len(metaVals)
			}

			return nil
		})
		return meta
	}
}

func getMeta() map[string][]string {
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{ReadOnly: true})
	defer db.Close()
	if err != nil {
		log.Fatal(err)
		return nil
	} else {
		meta := make(map[string][]string)
		db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("meta"))
			v := b.Get([]byte("names"))
			metaNames := make([]string, 0)
			json.Unmarshal(v, &metaNames)

			for _, name := range metaNames {
				vals := b.Get([]byte(name))
				metaVals := make([]string, 0)
				json.Unmarshal(vals, &metaVals)
				meta[name] = metaVals
			}
			return nil
		})
		return meta
	}
}

/**********************************************************************
	* Returns set of sample ids that match the parameters given
	* Empty set means no matches
	* nil means query was empty, and all samples are still available
***********************************************************************/
func filterSamples(db *bolt.DB, query map[string][]string) mapset.Set {

	var samples mapset.Set
	samples_empty := true

	db.View(func(tx *bolt.Tx) error {
		meta_samples := tx.Bucket([]byte("meta_samples"))
		meta := tx.Bucket([]byte("meta"))

		var meta_names []string
		meta_names_json := meta.Get([]byte("names"))
		json.Unmarshal(meta_names_json, &meta_names)

		if len(query) > 0 { // if there are filters selected
			for metaType, values := range query {
				metaTypeSamples := mapset.NewSet()
				for _, value := range values {
					valueSamples := meta_samples.Get([]byte(metaType + "_" + value))
					var samplesSlice []interface{}
					json.Unmarshal(valueSamples, &samplesSlice)
					samplesSet := mapset.NewSetFromSlice(samplesSlice)
					metaTypeSamples = metaTypeSamples.Union(samplesSet)
				}
				numSamplesFound := len(metaTypeSamples.ToSlice())
				if samples_empty && numSamplesFound > 0 {
					samples = metaTypeSamples
					samples_empty = false
				} else if !samples_empty && numSamplesFound > 0 {
					samples = samples.Intersect(metaTypeSamples)
				} else {
					samples = mapset.NewSet()
				}
			}
		}
		return nil
	})

	return samples

}

/**********************************************************************
	* Returns a map of datatypes which have maps from each available
	* value to the number of samples that have that value. Ex:
	{
		metaType1: {
			value1: 12, // 12 samples have this value
			value2: 9,
			value3 900
		},
		metaType2: {
			etc...
		}
	}
***********************************************************************/
func getFilteredMeta(query map[string][]string) map[string]map[string]int {
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{ReadOnly: true})
	defer db.Close()
	if err != nil {
		log.Fatal(err)
		return nil
	} else {
		filteredSamples := filterSamples(db, query)

		meta := make(map[string]map[string]int)
		db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("meta"))
			meta_samples := tx.Bucket([]byte("meta_samples"))
			v := b.Get([]byte("names"))
			metaNames := make([]string, 0)
			json.Unmarshal(v, &metaNames)

			for _, name := range metaNames {
				if name != "genes" {
					meta[name] = make(map[string]int)
					vals := b.Get([]byte(name))
					metaVals := make([]string, 0)
					json.Unmarshal(vals, &metaVals)
					for _, metaVal := range metaVals {
						valueSamples := meta_samples.Get([]byte(name + "_" + metaVal))
						var metaValSamples []interface{} // all samples that have this value for this metadata type
						json.Unmarshal(valueSamples, &metaValSamples)

						if filteredSamples != nil {
							metaValSamples = filteredSamples.Intersect(mapset.NewSetFromSlice(metaValSamples)).ToSlice()
						}
						if len(metaValSamples) > 0 {
							meta[name][metaVal] = len(metaValSamples)
						}
					}
				}
			}
			return nil
		})
		return meta
	}
}

/*
  returns two arrays of strings
  the first is a list of all the samples to get
  the second is a list of all properties of those samples to get
*/
func parseQuery(db *bolt.DB, query map[string][]string) ([]string, []string) {

	var samples mapset.Set
	samples_empty := true

	properties := make([]string, 1)
	properties[0] = "id"

	db.View(func(tx *bolt.Tx) error {
		meta_samples := tx.Bucket([]byte("meta_samples"))
		meta := tx.Bucket([]byte("meta"))

		for metaType, values := range query {
			metaTypeSamples := mapset.NewSet()
			for _, value := range values {
				valueSamples := meta_samples.Get([]byte(metaType + "_" + value))
				var samplesSlice []interface{}
				json.Unmarshal(valueSamples, &samplesSlice)
				samplesSet := mapset.NewSetFromSlice(samplesSlice)
				metaTypeSamples = metaTypeSamples.Union(samplesSet)
			}
			numSamplesFound := len(metaTypeSamples.ToSlice())
			if samples_empty && numSamplesFound > 0 {
				samples = metaTypeSamples
				samples_empty = false
			} else if !samples_empty && numSamplesFound > 0 {
				samples = samples.Intersect(metaTypeSamples)
			}
		}

		var meta_names []string
		meta_names_json := meta.Get([]byte("names"))

		json.Unmarshal(meta_names_json, &meta_names)

		sort.Strings(meta_names)

		for _, val := range meta_names {
			if val != "genes" {
				properties = append(properties, val)
			}
		}

		_, genesExist := query["genes"]
		if genesExist && len(query["genes"]) > 0 {
			for _, gene := range query["genes"] {
				properties = append(properties, gene)
			}
		} else {
			var all_genes []string
			genes_json := meta.Get([]byte("genes"))
			json.Unmarshal(genes_json, &all_genes)
			sort.Strings(all_genes)
			for _, gene := range all_genes {
				properties = append(properties, gene)
			}
		}

		return nil
	})

	samplesSlice := make([]string, 0)
	if samples != nil {
		for _, sample := range samples.ToSlice() {
			samplesSlice = append(samplesSlice, fmt.Sprint(sample))
		}
	}

	return samplesSlice, properties

}

func search(query string, done chan bool, genes *[]string) {
	searchResults := searchIndex.Lookup([]byte(query), 10)

	for _, index := range searchResults {
		prevDelim := index
		postDelim := index

		for true {
			if prevDelim > 0 && searchIndex.Bytes()[prevDelim-1] == delim {
				break
			}
			prevDelim--
		}

		for true {
			if postDelim < len(searchIndex.Bytes()) && searchIndex.Bytes()[postDelim] == delim {
				break
			}
			postDelim++
		}
		*genes = append(*genes, string(searchIndex.Bytes()[prevDelim:postDelim]))
	}
	fmt.Println(genes)
	done <- true
}
