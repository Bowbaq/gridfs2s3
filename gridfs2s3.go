package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"gopkg.in/mgo.v2"
)

var (
	access_key string
	secret_key string
	region     string
	bucket     string

	mongo_url  string
	mongo_db   string
	mongo_coll string

	workers int
)

func init() {
	flag.StringVar(&access_key, "k", "", "AWS access key")
	flag.StringVar(&secret_key, "s", "", "AWS secret key")
	flag.StringVar(&region, "r", "us-east-1", "AWS region")
	flag.StringVar(&bucket, "b", "", "S3 bucket for the files")

	flag.StringVar(&mongo_url, "h", "mongodb://localhost", "MongoDB connection string (e.g. mongodb://host1:port1,host2:port2)")
	flag.StringVar(&mongo_db, "d", "", "MongoDB database name")
	flag.StringVar(&mongo_coll, "c", "", "Prefix of MongoDB collection to migrate. Default is to migrate everything. Use full name to migrate a single collection")

	flag.IntVar(&workers, "w", 1, "Number of parallel workers. 2 x GOMAXPROCS seems to work well")
}

func main() {
	flag.Parse()
	check_args()

	fmt.Println("Preparing S3 client")
	auth, err := aws.GetAuth(access_key, secret_key)
	check(err)
	client := s3.New(auth, aws.Regions[region])
	bucket := client.Bucket(bucket)

	fmt.Println("Getting the list of existing keys")
	existing, err := bucket.GetBucketContents()
	check(err)

	fmt.Println("Preparing MongoDB client")
	session, err := mgo.Dial(mongo_url)
	check(err)
	db := session.DB(mongo_db)

	fmt.Println("Getting list of image collections")
	collections, err := db.CollectionNames()
	check(err)

	// Migrate files
	for _, collection := range collections {
		// Skip non-files collections
		if !strings.HasSuffix(collection, "files") {
			continue
		}
		// Use user specified filter if provided
		if mongo_coll != "" && !strings.HasPrefix(collection, mongo_coll) {
			continue
		}

		prefix := strings.TrimSuffix(collection, ".files")
		gfs := db.GridFS(prefix)
		files := gfs.Find(nil).Iter()

		total_count, _ := gfs.Find(nil).Count()
		fmt.Println("Migrating", total_count, "images for", prefix)

		migrated := make(chan int, workers)
		wg := new(sync.WaitGroup)
		wg.Add(workers)

		for i := 0; i < workers; i++ {
			go func(id int) {
				var f *mgo.GridFile
				file_count := 0

				for gfs.OpenNext(files, &f) {
					path := prefix + "/" + f.Name()

					if _, exists := (*existing)[path]; exists {
						log.Println(id, "SKIPPING", path)
						file_count++
						continue
					}

					log.Println(id, "INSERTING", path)
					err := bucket.PutReader(path, f, f.Size(), f.ContentType(), s3.BucketOwnerFull)
					if err != nil {
						log.Println(id, "INSERT ERROR", err)
						continue
					}

					file_count++
					if file_count%100 == 0 {
						migrated <- 100
					}
				}

				wg.Done()
			}(i)
		}

		done := make(chan bool)
		go func() {
			migrated_count := 0
			for m := range migrated {
				migrated_count += m
				if migrated_count%1000 == 0 {
					fmt.Println("Migrated", migrated_count, "/", total_count, "images for", prefix)
				}
			}
			fmt.Println("Migrated", total_count, "/", total_count, "images for", prefix)
			done <- true
		}()

		wg.Wait()
		close(migrated)
		<-done

		check(files.Close())
	}
}

func check_args() {
	if access_key == "" || secret_key == "" {
		log.Fatal("AWS credentials are required")
	}
	if _, ok := aws.Regions[region]; !ok {
		log.Fatal("Invalid region name")
	}
	if bucket == "" {
		log.Fatal("Bucket name is require")
	}
	if mongo_url == "" {
		log.Fatal("MongoDB connection string is required")
	}
	if mongo_db == "" {
		log.Fatal("MongoDB database name is required")
	}
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
