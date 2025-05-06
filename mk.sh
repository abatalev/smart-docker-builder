#!/bin/sh
LINTER="2.1.6"
CDIR=$(pwd)

export CGO_ENABLED=0

OS_NAME=$(cat /etc/os-release | awk -F= '/^NAME=/{ print $2; }')
OS_VERSION=$(cat /etc/os-release | awk -F= '/^VERSION_ID=/{ print $2; }')

GO_VERSION=$(gawk '/^go/{ print $2; }' ./go.mod)
GO_INSTALLED=$(go version| gawk '{print $3; }')
GOBIN="go${GO_VERSION}"

echo "### -[*]-[ Gathering Facts ]------------"
echo "### os ${OS_NAME} ${OS_VERSION}"
echo "### required go version go${GO_VERSION}"
echo "### installed go version ${GO_INSTALLED}"

if [ ! -d build ]; then
    mkdir build
fi

cd $CDIR || exit
if [ ! -f "build/golangci-lint" ]; then
  echo "### install golangci-lint $LINTER"
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b build/ "v$LINTER"

  if ! build/golangci-lint --version; then
	  echo "### aborted install golangci-lint"
	  exit 1
  fi
  echo "### done install golangci-lint"
else 
  # check golangci-lint v1
  if ! build/golangci-lint --version --format short 2> /dev/null; then
	echo "### failed check golangci-lint v1"

    # check golangci-lint v2
    if ! build/golangci-lint version --short 1> /dev/null; then 
	  echo "### failed check golangci-lint v2"
	  echo "### aborted check golangci-lint"
	  exit 1
    fi
	echo "### checked golangci-lint v2"
    XLINTER=$(build/golangci-lint version --short)
  else 
	echo "### checked golangci-lint v1"
    XLINTER=$(build/golangci-lint --version --format short)
  fi
  echo "### golangci-lint installed ${XLINTER}"

  if [ "$XLINTER" != "$LINTER" ]; then
    rm build/golangci-lint
    echo "### reinstall golangci-lint $LINTER"
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b build/ "v$LINTER"

    if ! build/golangci-lint --version; then
		echo "### aborted reinstall golangci-lint"
		exit 1
	fi
    echo "### done reinstall golangci-lint"
  fi  
fi

GO_BINDATA=$(which go-bindata) 
if [ "$GO_BINDATA" = "" ]; then
    go install github.com/go-bindata/go-bindata/go-bindata@latest
fi

if [ ! -f build/prj2hash ]; then
    echo "### -[*]-[ install prj2hash ]------------"
    cd build || exit
    # git clone https://github.com/abatalev/prj2hash prj2hash.git
    git clone http://localhost:3000/andrey/prj2hash prj2hash.git
    cd prj2hash.git || exit
	
	go mod tidy
	go build .

    cp prj2hash ../
    cd ${CDIR}/build || exit
    rm -f -R prj2hash.git
    echo "### done build tools"
fi

cd "$CDIR" || exit
if [ ! -f "build/gototcov" ]; then
    cd build || exit
    git clone https://github.com/jonaz/gototcov gototcov.git
    cd gototcov.git || exit
    go get golang.org/x/tools/cover
    go build .
    cp gototcov.git ../gototcov
    cd "${CDIR}/build" || exit
    rm -f -R gototcov.git
    echo "### done build tools"
fi

cd "$CDIR" || exit
# echo "### -[*]-[ Mod ]------------"
# go mod tidy

echo "### -[*]-[ Generate ]--------"
if ! go generate; then
    echo "### aborted"
    exit 1
fi

cd "${CDIR}" || exit
echo "### -[*]-[ Lint ]------------"
if ! ./build/golangci-lint run ./...; then
    echo "### aborted"
    exit 1
fi

echo "### -[*]-[ Test ]------------"
if ! go test -v -coverpkg=./... -coverprofile=coverage.out ./... > /dev/null; then
    echo "### aborted"
    exit 1
fi

echo "### total coverage"
if ! ./build/gototcov -f coverage.out -limit 80; then
    echo "### open browser"
    go tool cover -html=coverage.out
    echo "### aborted"
    exit 1
fi

echo "### -[*]-[ Mutating tests ]------------"
if ! ~/go/bin/gremlins unleash; then
    echo "### aborted"
    exit 1
fi

cd "$CDIR" || exit

echo "### -[*]-[ Build ]------------"
if ! go build -o sdb .; then
    echo "### aborted"
    exit 1
fi

build_app_git(){
    GIT_HASH=$1
    if [ -f "./build/prj2hash" ]; then
        P2H_HASH=$(./build/prj2hash)
    fi
    go build -ldflags "-X main.gitHash=${GIT_HASH} -X main.p2hHash=${P2H_HASH}" -o sdb .
}

echo "### build application with version"
build_app_git "$(git rev-list -1 HEAD)"

echo "### -[*]-[ Show version ]------------"
./sdb -version
echo "### -[*]-[ Show help ]------------"
./sdb -help
echo "### -[*]-[ Launch examples ]------------"
for i in examples/Dockerfile.*;
do
    echo "==> test $i"
    ./sdb "$i"
done
echo "### -[*]-[ The End ]------------"
