package main

import (
	"encoding/json"
	"log"

	"fmt"

	"os"

	"github.com/boltdb/bolt"
	"github.com/deckarep/golang-set"
)

func main() {
	queryMap := map[string][]string{}
	// queryMap["SM_Name"] = make([]string, 1)
	// queryMap["SM_Name"][0] = "neratinib"

	// fmt.Println(queryMap)

	fmt.Println(getFilteredMeta(queryMap))

	queryMap["SM_Name"] = make([]string, 1)
	queryMap["SM_Name"][0] = "neratinib"

	fmt.Println(getFilteredMeta(queryMap))

}

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

func getFilteredMeta(query map[string][]string) map[string]map[string]int {
	db, err := bolt.Open(os.Args[1], 0600, &bolt.Options{ReadOnly: true})
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
