# Smart Docker Build

## Intro

- Build a Docker image using only the Dockerfile. 
- Tag calculation based on Dockerfile data.
- Build only after finding changes. 
- Gathering facts after assembly.
- Create and publish tags using the collected facts. 

## Usage

```sh
$./sdb examples/Dockerfile.example
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
