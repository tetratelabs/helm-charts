#/bin/sh

VERSION=$(echo $1 | sed -e 'sXupdate-istio-XX')

if [ -d $VERSION ]; then
    echo "Version $VERSION must be specified"
    exit 1
fi

HERE=$(pwd)

if [ ! -d $HERE/charts/istio/$VERSION ]; then
    echo "Version $VERSION must exist"
    exit 1
fi

# $HERE/build contains the tarballs
# $HERE/pages contains the index.yaml

if [ -n "GITHUB_OUTPUT" ]; then
    echo "INDEX=$HERE/index.yaml" >> "$GITHUB_OUTPUT"
    echo "VERSION=$VERSION" >> "$GITHUB_OUTPUT"
    echo "BUILDDIR=$HERE/build" >> "$GITHUB_OUTPUT"
fi

ls -lR $TMPDIR
