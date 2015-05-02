package main

import (
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/s3"
	"github.com/awslabs/aws-sdk-go/service/s3/s3manager"
	"log"
	"os"
)

func main() {
	bucket := os.Args[1]
	dest := os.Args[2]
	path := os.Args[3]

	s3 := s3.New(&aws.Config{Region: "us-west-2"})
	log.Printf("Uploading to bucket=%s key=%s with contents of %s", bucket, dest, path)

	file, err := os.Open(path)

	if err != nil {
		log.Fatal(err)
	}

	upload := s3manager.UploadInput{
		Body:   file,
		Bucket: &bucket,
		Key:    &dest,
	}

	uploadResult, err := s3manager.Upload(s3, &upload, nil)

	log.Printf("%s", path)
	log.Printf("%v", uploadResult)
}
