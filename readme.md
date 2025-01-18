# Smart Docker Build

## Intro

- Build a Docker image using only the Dockerfile. 
- Tag calculation based on Dockerfile data.
- Build only after finding changes. 
- Gathering facts after assembly.
- Create and publish tags using the collected facts. 

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
