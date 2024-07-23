
# Env manager v2

A environment variable manager for AWS Amplify, DGO Apps and OCI Object Storage, initially designed for Collection

## Installation

The env manager v2 can be installed with Brew using the following command:

```bash
brew tap stanyzra/homebrew-env-manager-v2
brew install stanyzra/env-manager-v2/homebrew-env-manager-v2
```

## Updating

Once already installed, the env manager v2 can be updated with Brew using the following command:

```bash
brew upgrade stanyzra/env-manager-v2/homebrew-env-manager-v2
```

## Configuring

To actual use the env manager v2, you must configure your AWS, DGO and OCI credentials first (only OCI available at the moment). Theses credentials must have permission for Amplify, DGO Apps and OCI Object Storage, since the software need to list, create, update and delete resources from these services.

To configure, run the following command:

```bash
env-manager-v2 configure
```