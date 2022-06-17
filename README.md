# Tetrate Istio Distribition(TID)  Helm Charts

Helm charts to deploy istio components.These charts defaults to multi-arch istio images which can be run  on x86 and Arm based Instances, included docker images support multiple architecture i.e single image can run on x86 or arm based processor.

## Setup Helm repo

1. Add helm repo 

```sh
helm repo add tetratelabs https://tetratelabs.github.io/helm-charts/
helm repo ls
```

2. Check available charts and versions

```sh
helm search repo tetratelabs
helm search repo tetratelabs --versions
```

## Istio installation

1. Create istio-system namespace for installing control-plane

```sh
kubectl create namespace istio-system
```

2. Install base chart which setups required clusterwide k8 resources needed for istiod.

```sh
helm install istio-base tetratelabs/base -n istio-system
// install specific version eg 1.13.3
helm install istio-base tetratelabs/base -n istio-system --version 1.13.3
```
3. To customize the installation or check the configuration option

```sh
// to inspect the values.yaml for particular chart
helm show values tetratelabs/istiod
// to check the  output of the chart in fully rendered Kubernetes resource templates
helm template tetratelabs/istiod
```

4. Install istiod chart which install istio control plane

```sh
helm install istiod tetratelabs/istiod -n istio-system
// install specific version eg 1.13.3
helm install istiod tetratelabs/istiod -n istio-system --version 1.13.3
```

5. Install Istio ingress and egress gateways (optional)

```sh
kubectl create namespace istio-ingress
helm install istio-ingress itetratelabs/istio-ingress -n istio-ingress
// install specific version eg 1.13.3
helm install istio-ingress itetratelabs/istio-ingress -n istio-ingress --version 1.13.3
```


## Istio installation with Istio Operator

Instead of using Helm to install Istio itself, user can use Helm to install the [Istio Operator][istio-operator], and then create `IstioOperator` resource(s) in  cluster to install Istio.

```sh
kubectl create namespace istio-operator
helm install istio-operator tetratelabs/istio-operator \
  --set operatorNamespace=istio-operator \
  --set watchedNamespaces=istio-system
```

[istio-operator]: https://istio.io/latest/docs/setup/install/operator/

## Uninstallation

1. List all helm releases.

```sh
helm ls -n istio-system
```

2. Delete  helm releases and associated namespaces.

```sh
helm uninstall istiod -n istio-system
helm uninstall istio-base -n istio-system
helm uninstall istio-ingress -n istio-ingress
kubectl delete namespace istio-ingress istio-system
```

## Customize helm charts.

Configuration options to customize installation by overridding defaults using --set <:> command in helm install command

| **Parameter**                     | **Description**                                 | **Values**              | **Default**                      |
|-----------------------------------|-------------------------------------------------|-------------------------|----------------------------------|
| global.hub                        | Specifies the HUB for most images used by Istio | registry/namespace      | containers.istio.tetratelabs.com |
| global.tag                        | Specifies the TAG for most images used by Istio | valid image tag         | 1.13.5-tetrate-multiarch-v1      |
| global.proxy.image                | Specifies the proxy image name                  | valid proxy name        | proxyv2                          |
| global.istioNamespace             | Specifies  the namespace for istio controlplane | valid namespace         | istio-system                     |
| global.imagePullPolicy            | Specifies the image pull policy                 | valid image pull policy | IfNotPresent                     |
| global.imagePullSecret            | ImagePullSecrets for all ServiceAccount         | Valid imagepullsecret   |                                  |
| global.defaultPodDisruptionBudget | pod disruption budget for the control plane     | bool                    | true                             |
