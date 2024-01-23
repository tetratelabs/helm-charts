To hack this repository, you need to have [Go](https://go.dev/doc/install) installed.
We use [mage](https://magefile.org/), as the build system.
Currently, we only have a single [magefile.go](./magefile.go).

To test running locally, you can run all of the "pack" tasks with `FORCED` and `DRY_RUN` environment variables set.

As an example:

```console
FORCED=1 DRY_RUN=1 mage packistio
```

Available `mage` targets are (`mage -l`):

```console
Targets:
  index         generates Helm charts index, and optionally merge with existing index.yaml.
  packAddons    packs addons Helm charts.
  packDemos     packs demos Helm charts.
  packIstio     packs versioned Istio Helm charts.
  packSystem    packs system Helm charts.
```
