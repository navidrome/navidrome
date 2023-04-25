# Kubernetes

A couple things to keep in mind with this manifest:

1. This creates a namespace called `navidrome`. Adjust this as needed.
1. This manifest was created on [K3s](https://github.com/k3s-io/k3s), which uses its own storage provisioner called [local-path-provisioner](https://github.com/rancher/local-path-provisioner). Be sure to change the `storageClassName` of the `PersistentVolumeClaim` as needed.
1. The `PersistentVolumeClaim` sets up a 2Gi volume for Navidrome's database. Adjust this as needed.
1. Be sure to change the `image` tag from `ghcr.io/navidrome/navidrome:0.49.3` to whatever the newest version is.
1. This assumes your music is mounted on the host using `hostPath` at `/path/to/your/music/on/the/host`. Adjust this as needed.
1. The `Ingress` is already configured for `cert-manager` to obtain a Let's Encrypt TLS certificate and uses Traefik for routing. Adjust this as needed. 
1. The `Ingress` presents the service at `navidrome.${SECRET_INTERNAL_DOMAIN_NAME}`, which needs to already be setup in DNS.
