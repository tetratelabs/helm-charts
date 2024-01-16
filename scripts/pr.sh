#/bin/sh

VERSION=$(echo $1 | sed -e 'sXrefs/heads/update-istio-XX')
URL=$2

if [ -d $VERSION ]; then
    echo "Version $VERSION must be specified"
    exit 1
fi

HERE=$(pwd)

if [ ! -d $HERE/charts/istio/$VERSION ]; then
    echo "Version $VERSION must exist"
    exit 1
fi

TMPDIR=$(mktemp -d)
trap "cd $HERE && rm -rf $TMPDIR" EXIT

ITEMS=$(ls -1 $HERE/charts/istio/$VERSION)

git checkout istio-$VERSION
git fetch origin 
git switch -t gh-pages;true
git switch gh-pages;true
cp -f index.yaml $TMPDIR
git checkout istio-$VERSION

git branch

cd $TMPDIR

for ITEM in $ITEMS; do
    if [ ! -f $HERE/charts/istio/$VERSION/$ITEM/Chart.yaml ]; then
        continue
    fi
    ISTIO_VERSION=$(cat $HERE/charts/istio/$VERSION/$ITEM/Chart.yaml| grep version: | awk '{print $2}')
    CHART_URL=$URL/istio-$VERSION/$ITEM-$ISTIO_VERSION.tgz
    LAST_ITEM=/$ITEM-$ISTIO_VERSION.tgz/
    echo "Processing $ITEM $ISTIO_VERSION $CHART_URL"
    helm package $HERE/charts/istio/$VERSION/$ITEM
    helm repo index  . --merge $TMPDIR/index.yaml --url $CHART_URL 
done

mkdir $HERE/build
cp -f $TMPDIR/*.tgz $HERE/build

sed -i "sX${LAST_ITEM}X/Xg" $TMPDIR/index.yaml

mkdir $HERE/pages
cp -f $TMPDIR/index.yaml $HERE/pages

cd $HERE

if [ -n "GITHUB_OUTPUT" ]; then
    echo "INDEX=$HERE/pages/index.yaml" >> "$GITHUB_OUTPUT"
    echo "VERSION=$VERSION" >> "$GITHUB_OUTPUT"
    echo "BUILDDIR=$HERE/build" >> "$GITHUB_OUTPUT"
fi

ls -lR $TMPDIR
