gridfs2s3 - A tool to migrate MongoDB GridFS files to AWS S3
============================================================

This is a simple tool that will grab all the files in the GridFS you point it to, and stick them in S3

Installation
------------

```bash
go install github.com/Bowbaq/gridfs2s3
```

Usage
-----
```bash
gridfs2s3 -h
flag needs an argument: -h
Usage of gridfs2s3:
  -b="": S3 bucket for the files
  -c="": Prefix of MongoDB collection to migrate. Default is to migrate everything. Use full name to migrate a single collection
  -d="": MongoDB database name
  -h="mongodb://localhost": MongoDB connection string (e.g. mongodb://host1:port1,host2:port2)
  -k="": AWS access key
  -r="us-east-1": AWS region
  -s="": AWS secret key
  -w=1: Number of parallel workers. 2 x GOMAXPROCS seems to work well
exit status 2
```

Example
-------
```bash
# Basic usage
gridfs2s3 -k $AWS_ACCESS_KEY -s $AWS_SECRET_KEY -r "eu-west-1" -b "bucket-for-files" -h "mongodb://123.123.123.123" -d "mongodb-database-name"

# Use 8 parallel workers to speed things up (2 * GOMAXPROCS works pretty well)
gridfs2s3 -k $AWS_ACCESS_KEY -s $AWS_SECRET_KEY -r "eu-west-1" -b "bucket-for-files" -h "mongodb://123.123.123.123" -d "mongodb-database-name" -w 8

# Filter which collections to migrate. This will migrate image_full.files, image_medium.files, but not fs.files (i.e. prefix match)
gridfs2s3 -k $AWS_ACCESS_KEY -s $AWS_SECRET_KEY -r "eu-west-1" -b "bucket-for-files" -h "mongodb://123.123.123.123" -d "mongodb-database-name" -c "image"
```