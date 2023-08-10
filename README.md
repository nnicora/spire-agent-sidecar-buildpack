# spire-agent-sidecar-buildpack
## Description
This custom Cloud Foundry Buildpack includes the following components to a deployment: 
- SPIRE agent 
- Envoy, only if it is enabled by the `SPIRE_ENVOY_PROXY` environment variable in the manifest.
## Requirements
Cloud Foundry space and an application to deploy with the buildpack.
## Usage
For Cloud Foundry to be able to pull the buildpack from 'github.tools.sap', the URL of the buildpack - including a personal access token granting read access to public repositories - has to be provided in the application's _manifest.yml_ (see also [CF Docs: Deploying With Custom Buildpacks](https://docs.cloudfoundry.org/buildpacks/custom.html#deploying-with-custom-buildpacks)) as follows:

```
buildpacks:
  - https://<USERNAME>:<PERSONAL_ACCESS_TOKEN>@github.tools.sap/pse/spire-agent-sidecar-buildpack#TAG
```

Hints:
- **USERNAME**: You can put any value here (it is seemingly not evaluated by GitHub).
- **PERSONAL_ACCESS_TOKEN**: Must be the value of a personal access token by github.tools.sap granting read access to public repos. Note: The token may be exposed as part of the logs of the CF app.
- **TAG**: Usage of a tag is optional, yet, it is recommended to point to a specific version of the buildpack by appending a tag identifier (e.g. using #v0.1.0)

It is also possible to use provide the `personal access token` via a `vars-file` - see [Add Variables to a Manifest](https://docs.cloudfoundry.org/devguide/deploy-apps/manifest-attributes.html#variable-substitution).

## Creating a Personal Access Token

To create a personal access token, follow the [instructions on the official GitHub Docs](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token) and make sure to select **no scopes**, so the token only allows reading of public repositories, and define as approriate expiration time. 


Furthermore, it is recommended to use 'personal access token' created for a GitHub machine user (that is registered with a DL of your dev team) instead of your 'personal user account' You can create a new GitHub machine user by following the instructions on the respective section of the [SAP GitHub FAQ](https://pages.github.tools.sap/github/faq/#what-about-service-users). There is a [Self-Service Tool](https://technical-user-management.github.tools.sap/) to create GitHub machine users and it advises you to use an e-mail address of a [Distribution List](https://profiles.wdf.sap.corp/) (VPN required).


## Sidecar versions

- Spire Agent - **1.6.0-dev-unk**


## How to Obtain Support
[Create an issue](https://github.tools.sap/pse/spire-agent-sidecar-buildpack/issues) in this repository if you find a bug or have questions about the content.
 
For additional support, ask the PSE team.

## Contributing
If you wish to contribute code, offer fixes or improvements, please send a pull request. Due to legal reasons, contributors will be asked to accept a DCO when they create the first pull request to this project. This happens in an automated fashion during the submission process. SAP uses [the standard DCO text of the Linux Foundation](https://developercertificate.org/).

## Development

Prerequisites:

- Go
- Make
- Docker

### Package the Buildpack

Run `make package` to package the buildpack as a `.zip` file.

### Release Process

1. Update the [VERSION](VERSION) file.
2. Use GitHub to create a release.

Note: versioning follows [semver](https://semver.org/).

## License
Copyright (c) 2023 SAP SE or an SAP affiliate company. All rights reserved.
