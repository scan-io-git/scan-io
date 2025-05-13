<p align="center">
    <picture>
      <img src="assets/text-logo-light.png" height="100" alt="Scanio logo"/>
    </picture>
  </a>
</p>
<h2 align="center">
  All-in-One Multitool for Enhanced Security
</h2>

## What is Scanio?

Scanio simplifies security scanning for organizations by combining multiple open-source and enterprise-grade scanners into a single, customizable solution. Designed for teams with limited budgets, it enables teams to secure code efficiently and cost-effectively. By unifying interfaces and eliminating the need to develop tools and approaches for security processes from scratch, Scanio helps improve code quality, supports compliance efforts, and strengthens applications against vulnerabilities.

## Key Features
- Unified Interface: Use multiple scanners (e.g., Semgrep, Bandit, Trufflehog, CodeQL) with consistent commands and flags, reducing the learning curve for security teams and developers.
- Containerized Deployment: Prepackaged with dependencies, plugins, and rule sets for quick and hassle-free setup.
- Comprehensive Integration Support: Scanio seamlessly handles tasks such as code cloning, managing pull requests, and uploading scan results across VCS platforms like GitHub, GitLab, and Bitbucket.
- Infrastructure Ready: Configure and deploy Scanio with ease, using custom rules, configurations, and plugins.
- Extensible and Flexible: Designed for security applications but easily extends to QA and DevOps via its plugin-based architecture.
- Advanced SARIF Integration: SARIF report patching to meet specific requirements for enhanced usability and transform SARIF data into accessible HTML reports with interactive elements like code snippets and links.
- Compliance Simplified: Streamlines security processes across development stages, reducing effort and investment.
- Scalability: Adaptable for small teams or large enterprises, providing flexibility for diverse security scanning needs.

## Supported Integrations 

<div align="center">
  <img src="assets/Integrations.svg">
</div>

## Usage Scenarios
Each of these scenarios can be supported by specialized rule sets crafted for specific purposes or tailored to individual projects.

**Ad hoc Scanning**<br>
Ideal for security teams and developers looking to perform spot checks or analyze specific pieces of code manually during:
- Scan code during development.
- Perform security audits.

**Automated Background Scanning**<br>
Identify vulnerabilities and secrets in the codebase as a periodic process.

**CI/CD Pipeline Scanning**<br>
Automatically scan new code changes during branch merges.


## Getting Started
### Quick Start
Run your first scan:
```
git clone https://github.com/juice-shop/juice-shop
cd juice-shop
docker run -it -v $(pwd):/data ghcr.io/scan-io-git/scan-io analyse --scanner semgrep /data
```
### Installation
1) Installation with Docker:
```
docker pull ghcr.io/scan-io-git/scan-io   
```

2) Build and run from source:
```
git clone https://github.com/scan-io-git/scan-io
cd scan-io
make build docker
```

## Documentation
Explore Scanio's comprehensive [documentation](docs/README.md), structured using the Diátaxis framework.  

The documentation covers everything you need to know, including tutorials, how-to guides, conceptual explanations, and technical references, to help you use and extend Scanio effectively.
