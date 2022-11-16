## oci-artifacts
Working Go examples on how to push and pull any artifact (file) to/from OCI registries ("container registries"),
using the [ORAS project library](https://oras.land/).

Check out [`push`](./push.go) and [`pull`](./pull.go).

**Disclaimer**:
This only works with OCI registries that support OCI artifacts.
You can find a list of them [here](https://oras.land/implementors/).

Usage:
```
go run . --files "data/dock.jpg,data/ship.jpg" --image "REGISTRY/USER/REPO:TAG" \
 --username "<USER>" --password "<PASSWORD>" --output "."
```
