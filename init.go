package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	srchx "github.com/abasse/libsrchx"
	"github.com/icrowley/fake"
)

func init() {
	var err error
	flag.Parse()

	runtime.GOMAXPROCS(*flagWorkers)

	store, err = srchx.NewStore(*flagEngine, *flagStoragePath)
	if err != nil {
		log.Fatal(err)
	}

	Jsonpath = *flagStoragePath + "/json_data/"
	StoreJson = *flagStoreJson

	if *flagImportJson == true {
		files, err := WalkMatch(Jsonpath, "*.json")
		if err != nil {
			panic(err)
		}

		for i, f := range files {
			log.Printf("[INFO]Import %v %s", i, f)
			p := strings.Split(f, "json_data/")
			n := strings.Split(p[1], "/")
			ndx, _ := store.GetIndex(n[0] + "/" + n[1])

			file, _ := ioutil.ReadFile(f)
			jsonMap := make(map[string]interface{})
			err := json.Unmarshal([]byte(file), &jsonMap)
			if err != nil {
				panic(err)
			}
			ndx.Put(jsonMap)
		}
		log.Printf("[INFO]JSON import completed")
	}

	if *flagGenFakeData > 0 {
		go func() {
			ndx, _ := store.GetIndex("test/fake")
			fake.SetLang("en")

			for i := 0; i < *flagGenFakeData; i++ {
				ndx.Put(map[string]interface{}{
					"full_name": fake.FullName(),
					"country":   fake.Country(),
					"brand":     fake.Brand(),
					"email":     fake.EmailAddress(),
					"ip":        fake.IPv4(),
					"industry":  fake.Industry(),
					"age":       rand.Intn(100),
					"salary":    rand.Intn(20) * 1000,
					"power":     rand.Intn(10),
					"family":    rand.Intn(10),
				})
				log.Printf("[INFO]Testdata entry %v", i)
			}
			log.Printf("[INFO]Testdata creation completed")
		}()
	}
}

func WalkMatch(root, pattern string) ([]string, error) {
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err != nil {
			return err
		} else if matched {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}
