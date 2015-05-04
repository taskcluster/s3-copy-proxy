# S3 Copy Proxy

Copy objects from a source s3 bucket to a destination s3 bucket.
Designed to allow region specific caching of a "source" bucket.

## Usage

This package is intended to be built as binary and used directly or via
the docker image included in this package (taskcluster/s3-copy-proxy).

## Configuration

Aside from the command line configs this package will use the following
environment variables:

  - `AWS_ACCESS_KEY_ID` (required)
  - `AWS_SECRET_ACCESS_KEY` (required)
  - `INFLUXDB_URL` (optional when present will be used to send metrics)

## How it works

The core of the problem we faced was the costs of transferring data
between regions (both in dollar value and in time). This proxy was
designed to serve the 80% needs of our taskcluster deployment which
means serving up a decent (but not huge number) of different keys across
regions. To achieve these we use the following principals:

 - On error redirect back to the source where-ever possible

 - Download and serve only one copy of a key from the source (the rest
   of the requests will wait and be redirected to the newly uploaded key
   in the target bucket OR redirected back to the source.)

### TODO

  - Additional metrics on resource consumption in addition to those
    metrics we have on states.

  - Document and test non s3 sources (this actually should work now)

  - Explore optimizing keys over 5gig (this will outright fail right
    now!)

## Deploying the Docker Image

 - Requires godep to be installed (and obviously a working docker install).
 - Requires go which compiles for linux (or has setup a cross compiler
   to do this)

```sh
# <name> is the docker name + tag to use.
./docker.sh <name>
```

This script will cross compile the go binary for linux and run the
docker build generating a docker image with the `<name>` you provide to
`./docker.sh`.

Once this is built you can push to the registry or play with the image
locally...

## Developing

This package is unusual in that it has both go and node based test
suite.. There is no real good reason for this aside from the lack of
proficiency by the author to write lots of good go tests.

To run the entire test suite you must current have the following:

  - Access to our mozilla-taskcluster AWS account (sorry this is lame!)
  - [Godep](https://github.com/tools/godep) installed
  - NodeJS installed with a moderately recent version (0.10 and up)

Then run `make test` to run the entire test suite. For the advanced you
may also directly invoke `godep go test` or `./node_modules/.bin/mocha`
for the node tests.

## LICENSE

Copyright 2015, Mozilla Foundation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
