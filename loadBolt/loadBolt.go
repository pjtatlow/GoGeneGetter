package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"text/scanner"
	"time"

	"github.com/boltdb/bolt"
	"github.com/deckarep/golang-set"
	"gopkg.in/cheggaaa/pb.v1"
)

var print = fmt.Println

type MetaData map[int]mapset.Set

func (md MetaData) AddKey(i int) {
	md[i] = mapset.NewSet()
}

func (md MetaData) ContainsKey(i int) bool {
	_, exists := md[i]
	return exists
}

type MetaSamples map[int]map[string][]string

func (ms MetaSamples) AddKey(i int) {
	ms[i] = make(map[string][]string)
}

func (ms MetaSamples) ContainsKey(i int) bool {
	_, exists := ms[i]
	return exists
}

func main() {
	fileName, dbString, MetaVals, id, ms := getArgs()

	f, _ := os.Open(fileName)
	numLines, _ := wc(f)
	f.Close()
	f, _ = os.Open(fileName)
	defer f.Close()
	var s scanner.Scanner
	s.Init(f)

	bar := pb.StartNew(numLines)

	header := readLine(&s)
	allGenes := getGenes(header, id, MetaVals)
	bar.Increment()
	lineNum := 1
	done := false

	db, err := bolt.Open(dbString, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var wg sync.WaitGroup

	for !done {
		line := readLine(&s)
		if len(line) == 0 { // check if we are done with the file
			done = true
		} else { // if not increment the lineNumber
			lineNum++

			if len(line) != len(header) { // if this line isn't the same length as the header then the input is bad and throw a fit
				print("Line", lineNum, "only has", len(line), "columns, but the header has", len(header), "columns. Exiting.")
				os.Exit(2)
			}

			// insert the line into the database
			wg.Add(1)
			go insertLine(line, header, id, db, MetaVals, bar, &wg, ms)

			// if we make it here we can assume the line is good.
		}
	}
	wg.Wait()
	bar.FinishPrint("All rows inserted. Loading MetaData.")

	metaErr := insertMeta(header, db, MetaVals, allGenes)

	if metaErr != nil {
		panic(metaErr)
	}

	metaErr = insertMetaSamples(header, db, ms)

	if metaErr != nil {
		panic(metaErr)
	}

	print("Done.")
}

func readLine(s *scanner.Scanner) []string {
	var line []string
	if s.Peek() == scanner.EOF {
		return line
	}
	var token []rune
	var char rune
	done := false
	for !done {
		char = s.Next()
		if char == '\n' {
			done = true
		} else if char == ',' {
			line = append(line, string(token))
			token = make([]rune, 0)
		} else {
			token = append(token, char)
		}
	}
	return line
}

func getArgs() (string, string, MetaData, int, MetaSamples) {
	file := flag.String("file", "", "File to load")
	db := flag.String("db", "", "Database to insert data into")
	meta := flag.String("meta", "", "Array of columns to use as metadata. First column is zero. Ex. \"[0,1,2,3]\"")
	id := flag.Int("id", 0, "The index of the column of IDs. Defaults to 0.")

	MetaVals := make(MetaData)

	ms := make(MetaSamples)

	flag.Parse()

	if len(flag.Args()) > 0 {
		print("Unknown arguments:", flag.Args())
		os.Exit(1)
	}

	if len(*file) == 0 { // check if file argument was given.
		print("No file given!")
		os.Exit(1)
	} else {
		if _, err := os.Stat(*file); os.IsNotExist(err) { // check if file exists
			print("Input File does not exist")
			os.Exit(1)
		}
	}

	if len(*db) == 0 { // check if database argument was given
		print("No databse given!")
		os.Exit(1)
	} else {
		if _, err := os.Stat(*db); !os.IsNotExist(err) { // make sure database does not exist
			print("Database already exists!")
			os.Exit(1)
		}
	}

	if len(*meta) == 0 { // make sure meta columns were specified
		print("No meta columns given!")
		os.Exit(1)
	} else { // if they were try to turn that into an array of ints
		var metaCols []int
		json.Unmarshal([]byte(*meta), &metaCols)
		if len(metaCols) == 0 { // if it couldn't turn it into an array of ints, throw up everywhere.
			print("Invalid Meta Array.")
			os.Exit(1)
		} else {
			for _, val := range metaCols {
				MetaVals.AddKey(val)
				ms.AddKey(val)
			}
		}
	}

	if MetaVals.ContainsKey(*id) { // make sure ID is not in meta
		print("ID cannot be part of MetaData!")
		os.Exit(1)
	}
	return *file, *db, MetaVals, *id, ms
}

func wc(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}

func getGenes(header []string, id int, MetaVals MetaData) []string {
	allGenes := make([]string, 0)
	for i, v := range header {
		if i != id && !MetaVals.ContainsKey(i) {
			allGenes = append(allGenes, v)
		}
	}
	return allGenes
}

func insertLine(line []string, header []string, id int, db *bolt.DB, MetaVals MetaData, bar *pb.ProgressBar, wg *sync.WaitGroup, ms MetaSamples) {
	defer wg.Done()
	err := db.Batch(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucket([]byte(line[id]))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}

		for index, val := range line {
			if MetaVals.ContainsKey(index) {
				MetaVals[index].Add(val)
				_, exists := ms[index][val]
				if !exists {
					ms[index][val] = make([]string, 0)
				}
				ms[index][val] = append(ms[index][val], line[id])
			}
			err2 := bucket.Put([]byte(header[index]), []byte(line[index]))
			if err2 != nil {
				return fmt.Errorf("insert %s:%s into bucket: %s", header[index], line[index], err2)
			}
		}

		return nil
	})
	if err != nil {
		panic(err)
	}
	bar.Increment()
}

func insertMeta(header []string, db *bolt.DB, MetaVals MetaData, genes []string) error {
	metaNames := make([]string, 1)
	metaNames[0] = "genes"
	openErr := db.Batch(func(tx *bolt.Tx) error {
		bucket, metaErr := tx.CreateBucket([]byte("meta"))
		if metaErr != nil {
			return fmt.Errorf("create bucket: %s", metaErr)
		}

		for index, set := range MetaVals {
			metaNames = append(metaNames, header[index])
			arr, _ := json.Marshal(set.ToSlice())
			err3 := bucket.Put([]byte(header[index]), arr)
			if err3 != nil {
				return fmt.Errorf("insert sample into meta: %s", err3)
			}
		}

		arr, _ := json.Marshal(metaNames)
		err3 := bucket.Put([]byte("names"), arr)
		if err3 != nil {
			return fmt.Errorf("insert sample into meta: %s", err3)
		}

		genesArr, _ := json.Marshal(genes)
		err4 := bucket.Put([]byte("genes"), genesArr)
		if err4 != nil {
			return fmt.Errorf("insert genes into mets: %s", err4)
		}

		return nil
	})
	return openErr
}

func insertMetaSamples(header []string, db *bolt.DB, ms MetaSamples) error {
	openErr := db.Batch(func(tx *bolt.Tx) error {
		bucket, metaErr := tx.CreateBucket([]byte("meta_samples"))
		if metaErr != nil {
			return fmt.Errorf("create bucket: %s", metaErr)
		}
		for index, metaVals := range ms {
			meta_type := header[index]
			for value, set := range metaVals {
				ids, _ := json.Marshal(set)
				// print(set, string(ids))
				err := bucket.Put([]byte(meta_type+"_"+value), ids)
				if err != nil {
					return fmt.Errorf("create bucket: %s", err)
				}
			}
		}
		return nil
	})
	return openErr
}
