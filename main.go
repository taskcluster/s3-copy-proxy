package main

import (
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"log"
	"os"
)

func main() {
	bucket := os.Args[1]
	dest := os.Args[2]
	sourceFile := os.Args[3]

	auth, err := aws.EnvAuth()
	if err != nil {
		log.Fatal(err)
	}
	client := s3.New(auth, aws.USWest2)
	log.Printf("Uploading to bucket=%s key=%s with contents of %s", bucket, dest, sourceFile)

	file, err := os.Open(sourceFile)
	if err != nil {
		log.Fatal(err)
	}

	stat, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	bucketClient := client.Bucket(bucket)
	bucketClient.PutReader(dest, file, stat.Size(), "text/plain", s3.PublicReadWrite)
}
