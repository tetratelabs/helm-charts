## Prerequisites

To hack this repository, you need to have [Go](https://go.dev/doc/install) installed.
We use [mage](https://magefile.org/), as the build system.
Currently, we only have a single [magefile.go](./magefile.go).

## Running locally

To test running locally, you can run all of the "pack" tasks with `FORCED` and `DRY_RUN` environment variables set.

As an example:

```console
FORCED=1 DRY_RUN=1 mage packIstio
```

> [!NOTE]
> You need to import or generate a valid GPG key, with `trustee@tetrate-istio-subscription.iam.gserviceaccount.com` as the key name.
> This key name is currently hardcoded.

Available `mage` targets are (`mage -l`):

```console
Targets:
  index         generates Helm charts index, and optionally merge with existing index.yaml.
  packAddons    packs addons Helm charts.
  packDemos     packs demos Helm charts.
  packIstio     packs versioned Istio Helm charts.
  packSystem    packs system Helm charts.
```
