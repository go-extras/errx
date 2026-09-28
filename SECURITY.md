# Security Policy

## Supported Versions

Security fixes are provided for the latest minor release line.

| Version | Supported          |
| ------- | ------------------ |
| 1.3.x   | :white_check_mark: |
| < 1.3   | :x:                |

## Reporting a Vulnerability

We take the security of `errx` seriously. If you believe you have discovered a vulnerability, please do **not** disclose it publicly via public GitHub issues, pull requests, or discussions.

Please report vulnerabilities confidentially through GitHub's [Private Vulnerability Reporting](https://github.com/go-extras/errx/security/advisories/new) (if enabled for this repository) or directly to the project maintainer via email at [denis@vojt.uk](mailto:denis@vojt.uk).

### Information to Include

When submitting a security report, please include:
- A clear description of the vulnerability and its potential impact
- Affected package(s) or functions (for example `core`, `json`, or `stacktrace`)
- Steps to reproduce or a minimal proof-of-concept example
- Environment details (Go version, operating system, architecture)
- Any proposed remediation or patch, if available

### Response Process

1. **Acknowledgment**: We aim to acknowledge receipt of your report within 48 hours.
2. **Investigation**: We will verify the report, assess its severity, and keep you informed of our progress.
3. **Remediation**: Once a fix is prepared and verified, we will release a patch release and publish a security advisory giving appropriate credit to the reporter.
