package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/deckarep/golang-set"
	"github.com/kataras/iris"
)

func main() {

	iris.StaticWeb("/static", "./static/", 1)

	iris.Post("/query", func(ctx *iris.Context) {
		ctx.Response.Header.Set("Content-Type", "text/csv")
		var query map[string][]string
		queryString := ctx.PostValuesAll()["query"][0]
		json.Unmarshal([]byte(queryString), &query)

		ctx.Response.Header.Set("Content-disposition", "attachment; filename=test.csv")
		// ctx.StreamWriter(stream)

		ctx.Stream(func(w *bufio.Writer) {

			db, err := bolt.Open(os.Args[1], 0600, &bolt.Options{ReadOnly: true})
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()
			samples, properties := parseQuery(db, query)
			fmt.Fprint(w, strings.Join(properties, ","))
			fmt.Fprint(w, "\n")

			db.View(func(tx *bolt.Tx) error {
				for _, sample := range samples {

					b := tx.Bucket([]byte(sample))
					for _, prop := range properties {
						v := b.Get([]byte(prop))
						fmt.Fprint(w, string(v)+",")
					}
					fmt.Fprint(w, "\n")
					if err := w.Flush(); err != nil {
						return err
					}
				}
				return nil
			})
		})

	})

	iris.Get("/numSamples", func(ctx *iris.Context) {
		// ctx.Write("testing")

		var query map[string][]string
		query_string := ctx.URLParams()["query"]
		json.Unmarshal([]byte(query_string), &query)

		db, err := bolt.Open(os.Args[1], 0600, &bolt.Options{ReadOnly: true})
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		samples, _ := parseQuery(db, query)
		numSamples := len(samples)
		ctx.Write(strconv.Itoa(numSamples))

	})

	iris.Get("/", func(ctx *iris.Context) {
		ctx.ServeFile("./index.html", false)
	})

	iris.Get("/meta", func(ctx *iris.Context) {
		ctx.StreamWriter(getMeta)
	})

	iris.Listen(":8080")
}

func getMeta(w *bufio.Writer) {
	db, err := bolt.Open(os.Args[1], 0600, &bolt.Options{ReadOnly: true})
	defer db.Close()
	if err != nil {
		log.Fatal(err)
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

		metaString, _ := json.Marshal(meta)
		fmt.Fprint(w, string(metaString))
		if err := w.Flush(); err != nil {
			return
		}
	}

}

// returns two arrays of strings
// the first is a list of all the samples to get
// the second is a list of all properties of those samples to get
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
