# Smart Docker Build

## Intro

- Build a Docker image using only the Dockerfile. 
- Tag calculation based on Dockerfile data.
- Build only after finding changes. 
- Gathering facts after assembly.
- Create and publish tags using the collected facts. 

## Usage

```sh
$ cat examples/Dockerfile.example 
FROM alpine:latest

$ ./sdb examples/Dockerfile.example
smart docker build
 -> file examples/Dockerfile.example
 --> (abatalev/example:47e00eaa) image exists. build skipped
 --> gathering facts
 ---> fact: os-name = alpine
 ---> fact: os-version = 3.21.0
 --> create tags
 ---> mask $os-name|-|@os-version
 ----> tag alpine-3
 ----> tag alpine-3.21
 ----> tag alpine-3.21.0

 $ docker image list | grep example
abatalev/example  47e00eaa       4048db5d3672  6 weeks ago  7.83MB
abatalev/example  alpine-3       4048db5d3672  6 weeks ago  7.83MB
abatalev/example  alpine-3.21    4048db5d3672  6 weeks ago  7.83MB
abatalev/example  alpine-3.21.0  4048db5d3672  6 weeks ago  7.83MB
``` 

## Build

```sh
go build -o sdb .
```

## Usage

```sh
sdb build/Dockerfile.example
```

# for develop

```sh
./mk.sh
```

# TO DO 

- remove prefix. added imageName
- added prefix like ghcr.io/
- added push
- facts ...
