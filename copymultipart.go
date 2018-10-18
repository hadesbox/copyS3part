package main

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/aws/session"
	"math"
	"os"
	"strconv"
	"time"
)

func calculate_limits(lowlimit int64, hilimit int64) string {
	var buffer bytes.Buffer
	buffer.WriteString("bytes=")
	str := strconv.FormatInt(lowlimit, 10)
	buffer.WriteString(str)
	buffer.WriteString("-")
	str = strconv.FormatInt(hilimit, 10)
	buffer.WriteString(str)
	return buffer.String()
}

func copyPart(params *s3.UploadPartCopyInput, partNumber int, client *s3.S3, notify chan<- s3.CompletedPart) {
	fmt.Println("STARTING", partNumber, *params.CopySourceRange)
	respUploadPartCopy, err1 := client.UploadPartCopy(params)
	if err1 != nil {
		fmt.Println("ERROR UploadPartCopy", err1)
		panic(err1)
	}
	fmt.Println("SUCCESSS CopyPartResult", partNumber, *respUploadPartCopy.CopyPartResult.ETag)
	notify <- s3.CompletedPart{ETag: aws.String(*respUploadPartCopy.CopyPartResult.ETag), PartNumber: aws.Int64(int64(partNumber))}
}

func copy_object(sourceBucketName string, sourcePrefix string, destBucketName string, destPrefix string) {
	sess, err := session.NewSession(&aws.Config{
	    Region: aws.String("eu-west-1")},
	)
	client := s3.New(sess)
	result, err := client.ListObjects(&s3.ListObjectsInput{Bucket: &sourceBucketName, Prefix: &sourcePrefix})
	if err != nil {
		fmt.Println("ERROR ListObjects", err)
		return
	}
	fmt.Println("size", *result.Contents[0].Size, "key", *result.Contents[0].Key)

	paramsMultipartUpload := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(destBucketName),
		Key:    aws.String(destPrefix),
	}

	total_size := *result.Contents[0].Size

	max_chunk_size := int64(1000000000)

	total_chunks := int(math.Floor(float64(total_size) / float64(max_chunk_size)))
	fmt.Println("total chunks", total_chunks)
	remainer_chunk := total_size % max_chunk_size
	fmt.Println("remainder bytes", remainer_chunk)
	resp, err := client.CreateMultipartUpload(paramsMultipartUpload)

	if err != nil {
		fmt.Println("ERROR CreateMultipartUpload", err)
		panic(err)
	}

	fmt.Println("uploadid", *resp.UploadId)

	outputChan := make(chan s3.CompletedPart)

	for i := 0; i <= total_chunks; i++ {
		var res string
		if i == 0 {
			res = calculate_limits(int64(i)*max_chunk_size, (int64(i)+1)*max_chunk_size)
		} else if i == total_chunks {
			res = calculate_limits((int64(i)*max_chunk_size)+1, (int64(i)*max_chunk_size)+remainer_chunk-1)
		} else {
			res = calculate_limits((int64(i)*max_chunk_size)+1, int64(i+1)*max_chunk_size)
		}
		fmt.Println(i, "uploading chunk", i, res)

		paramsUploadInput := &s3.UploadPartCopyInput{
			Bucket:          aws.String(destBucketName),
			CopySource:      aws.String(sourceBucketName + "/" + sourcePrefix),
			CopySourceRange: aws.String(res),
			Key:             aws.String(destPrefix),
			PartNumber:      aws.Int64(int64(i + 1)),
			UploadId:        aws.String(*resp.UploadId),
		}

		go copyPart(paramsUploadInput, i, client, outputChan)

	}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(600 * time.Second)
		timeout <- true
	}()

	done_chunks := 0

	unord_parts := make([]string, total_chunks+1)

	for done_chunks <= total_chunks {
		select {
		case mensajito := <-outputChan:
			done_chunks++
			fmt.Println(*mensajito.PartNumber, *mensajito.ETag, done_chunks, total_chunks)
			unord_parts[*mensajito.PartNumber] = *mensajito.ETag
		case <-timeout:
			fmt.Println("Error, time out for copy time...")
		}
	}

	ord_parts := make([]*s3.CompletedPart, total_chunks+1)

	for key := range unord_parts {
		ord_parts[key] = &s3.CompletedPart{ETag: aws.String(unord_parts[key]), PartNumber: aws.Int64(int64(key + 1))}
	}

	paramsCompleteMultipartUploadInput := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(destBucketName),
		Key:      aws.String(destPrefix),
		UploadId: aws.String(*resp.UploadId),
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: ord_parts,
		},
	}
	_, err2 := client.CompleteMultipartUpload(paramsCompleteMultipartUploadInput)
	if err2 != nil {
		panic(err2)
	}
	fmt.Println("SUCCESS CompleteMultipartUpload")

}

func main() {

	argsWithoutProg := os.Args[1:]

	if len(argsWithoutProg) < 4 {
		fmt.Println("Error, not enough parameters usage:")
		fmt.Println("copymultipart <source_bucket> <source_key> <dest_bucket> <dest_key>")
		os.Exit(-1)
	}

	copy_object(argsWithoutProg[0], argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3])

}
