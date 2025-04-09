
# Env manager v2

A environment variable manager for AWS Amplify, DGO Apps and OCI Object Storage, initially designed for Collection. If you're using AWS Amplify or DGO Apps, the environment variables are stored in the cloud provider's console. For OCI Object Storage, the environment variables are stored in a bucket. This tool allows you to manage these environment variables in a single place, making it easier to update them, no matter where they are stored. We recommend using AWS Amplify and DGO Apps for front-end serverless applications and OCI Object Storage for back-end applications (specially with Kubernetes).

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

## Configuration

To use the **Env Manager v2**, you need to configure your AWS, DigitalOcean (DGO), and Oracle Cloud Infrastructure (OCI) credentials. These credentials must have permissions to manage resources in:
- **AWS Amplify**
- **DigitalOcean Apps**
- **OCI Object Storage**

The configuration file follows the **INI format** and is located at:
```
~/.env-manager-v2/config
```
Each cloud provider must be configured in separate sections, as shown in the example below:

```ini
[AWS]
aws_access_key_id = YOUR_AWS_ACCESS_KEY_ID
aws_secret_access_key = YOUR_AWS_SECRET
region = YOUR_AWS_REGION

[OCI]
region = YOUR_OCI_REGION
key_file = /path/to/oci_api_key.pem
namespace = YOUR_OCI_NAMESPACE
user = YOUR_OCI_USER_OCID
fingerprint = YOUR_OCI_API_KEY_FINGERPRINT
tenancy = YOUR_OCI_TENANCY_OCID
bucket_name = YOUR_OCI_BUCKET_NAME

[DGO]
dgo_api_token = YOUR_DIGITALOCEAN_API_TOKEN
```

### Configuration Sections
Below is a detailed list of available configuration options.

#### **[OCI] - Oracle Cloud Infrastructure**
| Key         | Description                    |
|-------------|--------------------------------|
| region      | OCI region                     |
| key_file    | Path to OCI SDK API key file   |
| namespace   | OCI namespace                  |
| user        | OCI user OCID                  |
| fingerprint | OCI API key fingerprint        |
| tenancy     | OCI tenancy OCID               |
| bucket_name | OCI bucket name                |

#### **[AWS] - Amazon Web Services**
| Key                   | Description            |
|-----------------------|------------------------|
| aws_access_key_id     | AWS access key ID      |
| aws_secret_access_key | AWS secret access key  |
| region                | AWS region             |

#### **[DGO] - DigitalOcean**
| Key           | Description            |
|---------------|------------------------|
| dgo_api_token | DigitalOcean API token |

#### **[DGO.APP_COMPONENTS] - DigitalOcean App Components**
| Key            | Description                            |
|----------------|----------------------------------------|
| app_components | Comma-separated list of app components |

#### **[K8S] - Kubernetes**
| Key                    | Description                       |
|------------------------|-----------------------------------|
| k8s_host               | Kubernetes API server URL         |
| k8s_token              | Kubernetes API token              |
| k8s_certificate_path   | Path to Kubernetes CA certificate |

#### **[ENVIRONMENTS] - Environments List**
| Key          | Description                                            |
|--------------|--------------------------------------------------------|
| environments | Default comma-separated list of available environments |

#### **[PROJECTS] - Projects List**
| Key       | Description                           |
|-----------|---------------------------------------|
| projects  | Comma-separated list of project names |

Each project has its own section containing environment-specific configurations:

```ini
["my-project"] # Don't forget to put the project name in quotes
dev.provider = AWS
dev.namespace = my-k8s-namespace
dev.configmap_name = my-k8s-configmap
dev.secret_name = my-k8s-secret
dev.branch_name = develop
prod.app_component_name = my-dgo-app-component-name
prod.app_name = my-dgo-app-name
```

We highly recommend using the **Github project** name as the project name in the configuration file. This way, you can easily identify the project and its environments.

#### **Project Configuration Keys**
| Key                                | Description                                                    |
|------------------------------------|----------------------------------------------------------------|
| `environments`                     | Comma-separated list of available environments for the project |
| `<environment>.provider`           | Cloud provider for the environment (e.g., AWS, OCI, DGO)       |
| `<environment>.namespace`          | Kubernetes Namespace (if applicable)                           |
| `<environment>.configmap_name`     | Kubernetes ConfigMap name (if applicable)                      |
| `<environment>.secret_name`        | Kubernetes Secret name (if applicable)                         |
| `<environment>.branch_name`        | GitHub branch name for the environment                         |
| `<environment>.app_component_name` | DigitalOcean App Component name (if applicable)                |
| `<environment>.app_name`           | DigitalOcean App name (if applicable)                          |

This structured configuration ensures flexibility and organization, allowing easy management of multiple environments and projects.

### Kubernetes integration

The **Env Manager v2** can also manage Kubernetes resources, such as ConfigMaps and Secrets. To do so, you need to provide the Kubernetes API server URL, a valid token, and the path to the CA certificate, as shown in the [Configuration file example](#configuration-file-example) section. We recommend using the [Kubernetes Reloader](https://github.com/stakater/Reloader) to automatically update the resources when the ConfigMap or Secret is updated.

In order to use the Kubernetes integration, you also need to create a Kubernetes service account with the necessary permissions to manage ConfigMaps and Secrets. A template for the Kubernetes service account and role is provided in the [manifests](manifests/permission-template.yml) directory. After creating the service account, you need to extract the token from the service account secret. The following command can be used to extract the token and the Kubernetes CA certificate:

```bash
kubectl get secret <sa_name>-secret -o jsonpath='{.data.token}' | base64 --decode
kubectl get cm kube-root-ca.crt -o jsonpath="{['data']['ca\.crt']}"
```

### Configuration file example

Your configuration file should look like this:

```ini
[AWS]
aws_access_key_id = YOUR_AWS_ACCESS_KEY_ID
aws_secret_access_key = YOUR_AWS_SECRET
region = us-east-1

[OCI]
region = us-ashburn-1
key_file = /path/to/oci_api_key.pem
namespace = my-oci-namespace
user = ocid1.user.oc1..example
fingerprint = 12:34:56:78:90:ab:cd:ef
tenancy = ocid1.tenancy.oc1..example
bucket_name = my-oci-bucket

[DGO]
dgo_api_token = my-digitalocean-token

[DGO.APP_COMPONENTS]
app_components = prod-app-component-name-big-proj, dev-app-component-name-small-proj, prod-app-component-name-small-proj

[K8S]
k8s_host = https://my-k8s-api-server
k8s_token = my-k8s-token
k8s_certificate_path = /path/to/ca.crt

[ENVIRONMENTS]
environments = dev,homolog,prod

[PROJECTS]
projects = my-big-front-end-project, my-small-front-end-project, my-backend-project-on-k8s

["my-big-front-end-project"]
environments = dev,homolog,prod
dev.provider = AWS
dev.branch_name = development
homolog.provider = AWS
homolog.branch_name = homologation
prod.provider = DGO
prod.branch_name = main
prod.app_component_name = prod-app-component-name-big-proj # required for DGO
prod.app_name = prod-app-name-big-proj # required for DGO

["my-small-front-end-project"]
environments = dev,prod
dev.provider = DGO
dev.branch_name = development
dev.app_component_name = dev-app-component-name-small-proj # required for DGO
dev.app_name = prod-app-name-small-proj # required for DGO
prod.provider = DGO
prod.branch_name = main
prod.app_component_name = prod-app-component-name-small-proj # required for DGO
prod.app_name = prod-app-name-small-proj # required for DGO

["my-backend-project-on-k8s"]
environments = dev,homolog,prod
dev.provider = OCI
dev.namespace = dev-namespace # required for Kubernetes integration
dev.configmap_name = dev-configmap # required for Kubernetes integration
dev.secret_name = dev-secret # required for Kubernetes integration
dev.branch_name = development
homolog.provider = OCI
homolog.namespace = homolog-namespace # required for Kubernetes integration
homolog.configmap_name = homolog-configmap # required for Kubernetes integration
homolog.secret_name = homolog-secret # required for Kubernetes integration
homolog.branch_name = homologation
prod.provider = OCI
prod.namespace = prod-namespace # required for Kubernetes integration
prod.configmap_name = prod-configmap # required for Kubernetes integration
prod.secret_name = prod-secret # required for Kubernetes integration
prod.branch_name = main
```

<!-- DEPRECATED, NEED TO UPDATE CONFIGURE COMMAND
To configure, run the following command:

```bash
env-manager-v2 configure
``` -->