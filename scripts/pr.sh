#/bin/sh

VERSION=$(echo $1 | sed -e 'sXupdate-istio-XX')
URL=$2
CURRENT_BRANCH=$1


if [ -d $VERSION ]; then
    echo "Version $VERSION must be specified"
    exit 1
fi

HERE=$(pwd)

if [ ! -d $HERE/charts/istio/$VERSION ]; then
    echo "Version $VERSION must exist"
    exit 1
fi

if [ -z "$GPG_SIGNER" ]; then
    echo "GPG_SIGNER must be specified"
    exit 1
fi

if [ -z "$GPG_PASSPHRASE" ]; then
    echo "GPG_PASSPHRASE must be specified"
    exit 1
fi

TMPDIR=$(mktemp -d)
trap "cd $HERE && rm -rf $TMPDIR" EXIT

ITEMS=$(ls -1 $HERE/charts/istio/$VERSION)

git checkout $CURRENT_BRANCH
git fetch origin 
git switch gh-pages
if [ -f index.yaml ]; then
    cp -f index.yaml $TMPDIR
fi

git checkout $CURRENT_BRANCH
git branch

cd $TMPDIR

for ITEM in $ITEMS; do
    if [ ! -f $HERE/charts/istio/$VERSION/$ITEM/Chart.yaml ]; then
        continue
    fi
    ISTIO_VERSION=$(cat $HERE/charts/istio/$VERSION/$ITEM/Chart.yaml| grep version: | awk '{print $2}')
    CHART_URL=$URL/release-istio-$VERSION/$ITEM-$ISTIO_VERSION.tgz
    LAST_ITEM=/$ITEM-$ISTIO_VERSION.tgz/
    echo "Processing $ITEM $ISTIO_VERSION $CHART_URL"
    echo $GPG_PASSPHRASE |helm package --sign $HERE/charts/istio/$VERSION/$ITEM --key $GPG_SIGNER --passphrase-file -
    if [ -f $TMPDIR/index.yaml ]; then
        helm repo index  . --merge $TMPDIR/index.yaml --url $CHART_URL 
    else
        helm repo index  . --url $CHART_URL 
    fi
done

mkdir $HERE/build
cp -f $TMPDIR/*.tgz $HERE/build
cp -f $TMPDIR/*.tgz.prov $HERE/build

sed -i "sX${LAST_ITEM}X/Xg" $TMPDIR/index.yaml

mkdir $HERE/pages
cp -f $TMPDIR/index.yaml $HERE/pages

cd $HERE

if [ -n "GITHUB_OUTPUT" ]; then
    echo "INDEX=$HERE/pages" >> "$GITHUB_OUTPUT"
    echo "VERSION=$VERSION" >> "$GITHUB_OUTPUT"
    echo "BUILDDIR=$HERE/build" >> "$GITHUB_OUTPUT"
fi

ls -lR $TMPDIR
