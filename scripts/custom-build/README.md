# Custom Scanio Deployment

This `Makefile` supports custom deployments of Scanio, including cases where users have their own versions of Scanio, plugins, and custom rule sets. The deployment process includes cloning the Scanio repository, applying custom rules, building rules, creating Docker images, and pushing them to a registry. The Makefile can be used in various environments, including as part of an internal CI/CD pipeline, local development, or version-controlled repositories.

## Prerequisites
Ensure that the following dependencies are available on your system before running the commands:
- `git`
- `docker`
- Python3 (for building the custom rule sets)

## Custom Deployment Workflow

1. **Custom Scanio Repository**:
    - You can override the default Scanio repository by setting the `SCANIO_REPO` variable in the command.
    - The repository will be cloned to the local directory specified in `SCANIO_REPO_DIR`.

2. **Custom Rule Set**:
    - By default, the `Makefile` will use `scanio_rules.yaml` located in the current directory. You can specify a custom rule set path via the `RULES_CONFIG` variable.
    - The rule set will be copied into the cloned repository in the `scripts/rules/` directory.
  
3. **Building Rules**:
    - Python dependencies will be installed in a virtual environment.

4. **Docker Image Build and Push**:
    - You can specify the Docker image version, target OS, and architecture using the `VERSION`, `TARGET_OS`, and `TARGET_ARCH` variables respectively.
    - Optionally, set the `REGISTRY` variable to specify where to push the Docker image.


## Makefile Targets

### 1. **help**: 
Displays the available targets and their descriptions.

```bash
make help
```

### 2. **clone-scanio-repo**: 
Clones the Scanio repository to the local directory defined in `SCANIO_REPO_DIR`.

```bash
make clone-scanio-repo SCANIO_REPO=<repo_url>
```

### 3. **copy-rules**:
Copies the custom rule set to the cloned Scanio repository.

```bash
make copy-rules RULES_CONFIG=<rules_path>
```

### 4. **build-rules**:
Builds the custom rule sets inside the cloned repository. This step includes setting up the Python environment by default and force overriding of the existed rules foled., if necessary.

```bash
make build-rules
```

### 5. **build-docker**:
Builds the Docker image for Scanio using the cloned repository. You can specify the Docker image version, OS, architecture, and registry.

```bash
make build-docker VERSION=<version> TARGET_OS=<os> TARGET_ARCH=<arch> REGISTRY=<registry>
```

### 7. **push-docker**:
Pushes the built Docker image to the registry. Ensure that you have specified the registry in the `REGISTRY` variable.

```bash
make push-docker VERSION=<version> REGISTRY=<registry>
```

### 8. **clean**:
Cleans up the cloned repository and generated files.

```bash
make clean
```

### 9. **build**:
Runs the full custom Scanio build process, including cloning, copying rules, building rules, building Docker images, and pushing the Docker image to the registry.

```bash
make build SCANIO_REPO=<repo_url> RULES_CONFIG=<rules_path> VERSION=<version> TARGET_OS=<os> TARGET_ARCH=<arch> REGISTRY=<registry>
```

## Example Usage

To build and deploy a custom version of Scanio:

```bash
make build SCANIO_REPO=https://github.com/your-org/custom-scanio.git RULES_CONFIG=custom_rules.yaml VERSION=2.0 TARGET_OS=linux TARGET_ARCH=amd64 REGISTRY=docker.io/your-username
```

This command will:
1. Clone the custom Scanio repository.
2. Copy the custom rule set.
3. Build the rules using the Python environment.
4. Build the Docker image.
5. Push the Docker image to `docker.io/your-username`.

## Things to Consider:
1. **Custom Plugins**: If you have custom plugins, they can be added to the `plugins/` directory in the cloned repository and will be built as part of the Scanio plugins.
2. **Customization**: You can modify the paths, Docker image versions, and other settings by overriding the default values with command-line variables.
