#!/usr/bin/env sh
set -ev

# constants
BINARY_NAME="kelvin"

buildTarget() {
  OS=$1
  ARCH=$2
  TARGET="$OS-$ARCH"

  if [[ $OS == "windows" ]]; then
    BINARY_NAME="$BINARY_NAME.exe"
  fi

  # Read latest git tag
  if git describe --abbrev=0 --tags; then
    GIT_TAG=$(git describe --abbrev=0 --tags)
  else
    GIT_TAG="unknown"
  fi

  # configure paths
  DIST_PATH=$GOPATH/dist
  OUTPUT_PATH=$DIST_PATH/"$BINARY_NAME-$TARGET-$GIT_TAG"
  OUTPUT_BINARY=$OUTPUT_PATH/$BINARY_NAME
  ARCHIVE_PATH=$DIST_PATH/archives
  ARCHIVE_NAME=$ARCHIVE_PATH/"$BINARY_NAME-$TARGET-$GIT_TAG"
  if [[ $OS == "windows" ]]; then
    ARCHIVE_NAME="$ARCHIVE_NAME.zip"
  else
    ARCHIVE_NAME="$ARCHIVE_NAME.tar.gz"
  fi
  mkdir -p $OUTPUT_PATH
  mkdir -p $ARCHIVE_PATH

  # Start go build
  echo ===== Starting build =====
  echo Target:          $BINARY_NAME $GIT_TAG $TARGET
  echo Output path:     $OUTPUT_PATH
  echo Output binary:   $OUTPUT_BINARY
  echo Output archive:  $ARCHIVE_NAME
  echo tar version:     $(tar --version)
  echo zip version:     $(zip --version)

  # test permissions
  touch $OUTPUT_BINARY
  touch $ARCHIVE_NAME

  export GOOS=$OS
  export GOARCH=$ARCH
  go build -ldflags "-X main.applicationVersion=${GIT_TAG}" -v -o $OUTPUT_BINARY

  # make binary executable
  $(chmod +x $OUTPUT_BINARY)

  # include license and readme
  $(cp README.md $OUTPUT_PATH/README.txt)
  $(cp LICENSE $OUTPUT_PATH/LICENSE.txt)

  # build archive
  if [[ $OS == "windows" ]]; then
    $(zip -r $ARCHIVE_NAME $OUTPUT_PATH)
  else
    $(tar cfvz $ARCHIVE_NAME $OUTPUT_PATH)
  fi
  echo ===== $TARGET build successfull =====
}

# MAIN
echo Start
buildTarget linux amd64
buildTarget linux 386
buildTarget linux arm
buildTarget freebsd amd64
buildTarget darwin amd64
buildTarget windows amd64
buildTarget windows 386
echo Done
