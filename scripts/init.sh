#!/bin/sh
# This script was used to get the helm charts from tetrate helm repo and convert them to the format that istio.io uses
# It is not used anymore, but kept here for reference
#
# Manual work was done to fix the charts that had discrepancies in the appVersion fields.
# The tarballs contains the charts that have different appVersion than the version in the index.yaml,
# the work was done by copying the charts from the github/helm-charts/{appVersion}
#
# The problematic charts versions were:
# - 1.16.6-tetrate-v0 (the tarball contains 1.16.6-tetrate-v3)
# - 1.18.5-tetrate-v0 (the tarball contains 1.18.5-tetrate-v2)

INDEX=https://raw.githubusercontent.com/tetratelabs/helm-charts/gh-pages/index.yaml

TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

START="1.16.6"
SED=sed
if [ "$(uname)" = "Darwin" ]; then
    SED=gsed
fi

function setup() {
    ITEMS="base:base cni:cni gateway:gateway istio-default:default istio-egress:gateways/istio-egress "
    ITEMS="$ITEMS istio-ingress:gateways/istio-ingress istio-operator:istio-operator istiod-remote:istiod-remote "
    ITEMS="$ITEMS istiod:istio-control/istio-discovery ztunnel:ztunnel"
    ITEMS="$ITEMS 1.18.2-tetrate-v0:1.18.2-tetrate-v1"
    for ITEM in $ITEMS; do
        K=`echo $ITEM | cut -d':' -f1`
        V=`echo $ITEM | cut -d':' -f2`
        mkdir -p $TMPDIR/map/$K
        echo $V > $TMPDIR/map/$K/V
    done
}

SKIP="1"
function process() {
    VERSION=`echo $1| cut -d'|' -f1`
    NAME=`echo $1 | cut -d'|' -f2`
    APPVERSION=`echo $1 | cut -d'|' -f3`
    APPVERSION_ORIG=$APPVERSION
    URL=`echo $1 | cut -d'|' -f4`
  
    if [ "x$VERSION" = "x$START" ]; then
        unset SKIP
    fi

    if [ "x$SKIP" = "x1" ]; then
        return
    fi

    case "$VERSION" in
    *-tetrate-v*) 
    HAS_TETRATE=1
    ;;
    *) 
    HAS_TETRATE=0
    ;;
    esac

    # Some charts have appVersion 1.0.0 (which must be wrong), but the tarball contains a different version
    if [ "x$APPVERSION" = "x1.0.0" ]; then
        if [ "x$HAS_TETRATE" = "x1" ]; then
            APPVERSION=$VERSION
        else
            APPVERSION=$VERSION-tetrate-v0

            if [ -d $TMPDIR/map/$APPVERSION ];then
                APPVERSION=$(cat $TMPDIR/map/$APPVERSION/V)
            fi
        fi
        
    fi

    ITEM=$(basename $URL)
    V=$(cat $TMPDIR/map/$NAME/V)
    DIR=charts/istio/$APPVERSION/$V

    DLDIR=$TMPDIR/dl
    rm -rf $DLDIR $DIR
    mkdir -p $DLDIR $DIR

    TGZ=$(basename $URL)
    curl -SsL -o $DLDIR/$TGZ $URL 

    EXTRACTED=$(tar tzf $DLDIR/$TGZ |cut -d'/' -f1|head -1)
    tar xzf $DLDIR/$TGZ -C $DLDIR
    mv $DLDIR/$EXTRACTED/* $DIR

    $SED -i "s|appVersion: $APPVERSION_ORIG|appVersion: $APPVERSION|g" $DIR/Chart.yaml
    echo "Processed $NAME $VERSION $APPVERSION $URL"
}

setup
ALL=$(curl -SsL $INDEX | yq  |  jq  '.entries[] | select(.. | objects | has("keywords") and (.keywords | any(. == "istio")))[] | .version + "|" + .name + "|" + .appVersion + "|" + .urls[0]'| tr -d '"'|sort|uniq)
for EACH in $ALL; do
  process $EACH
done

