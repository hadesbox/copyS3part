# What is this

So we wrote a AWS lambda function to move a file, from one bucket to another using the *copy* service in the API. But one day we notice an error mainly because of the file was bigger than 5 GB (the max file size for a non multipart copy operation) but also do the fact that even if we used AWS CLI (boto/python) on the lambda function, the lamda invoke would expire as all lambda functions have a maximum (hard) timeout of 60 seconds... and sometimes moving a file 10GB - 20GB took more or about 60 seconds. 

```
[luix@boxita copyMultipartS3]$ time aws s3 cp s3://sourcebucket/../.../...txt.gz s3://destbucket/../.../...txt.gz
copy: s3://sourcebucket/../.../...txt.gz
1093 parts...

real	0m63.182s
user	0m3.340s
sys	0m0.163s
[luix@boxita copyMultipartS3]$
```

I know that lambda functions are meant to execute small tasks, but I wanted to push the limits on the API and test if its possible to move big files within or less than 60 seconds with a lambda fucntion, so I've started and wrote my on program using AWS SDK for Node (because of its async nature) and go (surprise... concurrency!!!!). 

There were not big differences between go and node programs, both were faster than the CLI, chunk size may affect times, but for the ones we tested this was not a problem (this may  different with files of  several GB of size)

| Chunk Size    | Execution time  |
| ------------- |:---------------:|
| 500 MB        | 0m26.114s       |
| 200 MB        | 0m24.045s       |
| 100 MB        | 0m19.301s       |
| 50 MB         | 0m18.001s       |
| 20 MB         | 0m28.296s       |
| 10 MB         | 0m24.790s       |

Following the official AWS S3 documentation, 

https://aws.amazon.com/blogs/aws/amazon-s3-multipart-upload/

here are the programs install process and usage.

#Installation

## AWS Credentials

First of all you need to be sure to have your credentials installed properlly, if you are using the CLI, you probably already have them installed. 

http://docs.aws.amazon.com/AWSSdkDocsJava/latest//DeveloperGuide/credentials.html

## Nodejs Install

If you want to edit the nodejs program, you can check the full SDK documentation [here](http://aws.amazon.com/sdk-for-node-js/)

To install the nodejs program, as a **global** CLI too, you need to have nodejs and npm installed on your system, then after cloning this repo you can execute a **global install** of the prorgam.

```
[luix@boxita]$ sudo npm install -g
/usr/bin/multipartcopy -> /usr/lib/node_modules/copymultipart/index.js
copymultipart@1.0.0 /usr/lib/node_modules/copymultipart
└── aws-sdk@2.1.42 (xmlbuilder@0.4.2, xml2js@0.2.8, sax@0.5.3)
[luix@boxita]$
```

this will create the command **multipartcopy** that can be used on your terminal..

```
[luix@boxita]$ multipartcopy bucketsource path/to/source/key targetsource paath/to/destination/key
Total size to copy 9153817104 bytes
SUCCESS createMultipartUpload z8uvfgcq....
Total chunks to upload 9
Remainder bytes last chunk 153817104
uploading chunk 0 bytes=0-1000000000
uploading chunk 1 bytes=1000000001-2000000000
uploading chunk 2 bytes=2000000001-3000000000
....
SUCCESS completeMultipartUpload
[luix@boxita]$
```

if you want to tweak the chunk size, there is a varible on line 17 of the index.js file where you can change this.

## Go Install

If you want to tweak or compile the program, you willneed to install go sdk for AWS

```
http://aws.amazon.com/sdk-for-go/
```

then you can build the program using

```
[luix@boxita]$ go build copymultipart.go
```

then you can directly use the binary (crossplatform... go rocks!).

```
[luix@boxita copyMultipartS3]$ ./copymultipart "source-bucket" "source/to/file" "target-bucket" "target/to/file"
size 9153817104 key
total chunks 9
remainder bytes 153817104
uploadid DW1XVuJraJ....
0 uploading chunk 0 bytes=0-1000000000
...
SUCCESSS CopyPartResult 4 "842e05c0503b3a195e81934881f5685b"
4 "842e05c0503b3a195e81934881f5685b" 10 9
SUCCESS CompleteMultipartUpload
[luix@boxita copyMultipartS3]$ 
```
