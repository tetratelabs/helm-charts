# Tetrate Istio Distribution Helm Charts

## Usage

[Helm](https://helm.sh) must be installed to use the charts.
Please refer to Helm's [documentation](https://helm.sh/docs/) to get started.

Once Helm is set up properly, add the repository as follows:

```console
helm repo add tetratelabs https://tetratelabs.github.io/helm-charts
```

You can then run `helm search repo tetratelabs` to see the charts.

> [!NOTE]
> To list down all versions, you need to provide the `-l` and `--devel` flags.
> It is recommended to migrate to the newer charts repo: https://tis.tetrate.io/charts.
