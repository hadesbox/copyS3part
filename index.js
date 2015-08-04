#!/usr/bin/env node

if(process.argv.length < 6){
  console.log("Error", "not enough parameters, usage:")
  console.log("multipartcopy <source_bucket> <source_key> <dest_bucket> <dest_key>")
  process.exit(-1)
}

var AWS = require('aws-sdk');

AWS.config.region = 'eu-west-1';

var s3 = new AWS.S3();

total_size = 0;
//currently set to 1 GB
max_chunk_size = 1000000000;

source_prefix = process.argv[3];
source_bucket = process.argv[2];

destination_key = process.argv[5];
destination_bucket = process.argv[4];

parts_etags = [];
done_chunks = 0;

var params = {
  Bucket: source_bucket,
  Prefix: source_prefix
};

s3.listObjects(params, function(err, data) {

  if (err) console.log("ERROR listObjects", err, err.stack);
  else {

    total_size = data.Contents[0].Size

    console.log("Total size to copy", total_size, "bytes")

    var params = {
      Bucket: destination_bucket,
      Key: destination_key
    };

    s3.createMultipartUpload(params, function(err, data) {

      if (err) console.log("ERROR createMultipartUpload", err, err.stack);
      else {

        console.log("SUCCESS createMultipartUpload", data.UploadId);

        total_chunks = Math.floor(total_size / max_chunk_size)
        console.log("Total chunks to upload", total_chunks)
        remainer_chunk = total_size % max_chunk_size
        console.log("Remainder bytes last chunk", remainer_chunk)

        for (i = 0; i <= total_chunks; i++) {

          chunk = 0;

          if (i == 0)
            chunk = "bytes=" + (i * max_chunk_size) + "-" + (((i + 1) * max_chunk_size))
          else if (i == total_chunks)
            chunk = "bytes=" + ((i * max_chunk_size) + 1) + "-" + ((i * max_chunk_size) + remainer_chunk - 1)
          else
            chunk = "bytes=" + ((i * max_chunk_size) + 1) + "-" + (((i + 1) * max_chunk_size))

          console.log("uploading chunk", i, chunk)

          var params = {
            Bucket: destination_bucket,
            CopySource: source_bucket + "/" + source_prefix,
            Key: destination_key,
            PartNumber: i + 1,
            UploadId: data.UploadId,
            CopySourceRange: chunk
          };

          (function(params, data, i) {
            s3.uploadPartCopy(params, function(err, dataPartCopy) {
              if (err) console.log("ERROR uploadPartCopy", i, err, err.stack); // an error occurred
              else {

                console.log("SUCCESS uploadPartCopy for chunk", i);

                parts_etags[i] = dataPartCopy.CopyPartResult.ETag;

                done_chunks++;

                checkIfDone(dataPartCopy.CopyPartResult.ETag, data.UploadId, i)

              }
            });
          })(params, data, i);

        }
      }

    });

  }

});

function checkIfDone(etag, upload_id, partid) {

  console.log("chunk", partid, "with ETag", etag)

  if (done_chunks > total_chunks) {

    console.log("All chunks done")

    etags_params = [];

    for (i = 0; i < parts_etags.length; i++) {

      etags_params.push({
        ETag: parts_etags[i],
        PartNumber: i + 1
      })

    }

    console.log(etags_params)

    var params = {
      Bucket: destination_bucket,
      Key: destination_key,
      UploadId: upload_id,
      MultipartUpload: {
        Parts: etags_params
      },
      RequestPayer: 'requester'
    };

    s3.completeMultipartUpload(params, function(err, data) {
      if (err) console.log("ERROR completeMultipartUpload", err, err.stack); // an error occurred
      else console.log("SUCCESS completeMultipartUpload", data); // successful response
    });

  }

}