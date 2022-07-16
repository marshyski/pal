# Security Policy

## Supported Versions

The pal project is currently under active development. The following table provides information about the status of each release branch:

| Version     | Supported          |
| ----------- | ------------------ |
| `v2025.*.*` | :heavy_check_mark: |

**Note:** We only provide security updates for the current year CalVer releases or the latest release. Older versions/release branches are not actively supported.

## Reporting a Vulnerability

**DO NOT CREATE A PUBLIC GITHUB ISSUE FOR SECURITY VULNERABILITIES.**

Instead, please report any security vulnerabilities directly to GitHub Advisories on the pal GitHub page here: https://github.com/marshyski/pal/security/advisories/new

**When reporting a vulnerability, please provide the following information:**

- **Description:** A clear and concise description of the vulnerability.
- **Steps to Reproduce:** Detailed steps to reproduce the vulnerability, including any necessary code snippets, environment setup, or specific inputs.
- **Affected Version(s):** The version(s) of the project affected by the vulnerability (if known).
- **Potential Impact:** An assessment of the potential impact of the vulnerability.
- **Possible Mitigation:** If you have any suggestions for mitigating the vulnerability, please include them.

**Vulnerability Handling Process:**

1.  **Acknowledgement:** We will acknowledge your report within 48 hours.
2.  **Verification:** We will investigate and verify the reported vulnerability.
3.  **Fix Development:** If the vulnerability is confirmed, we will develop a fix as soon as possible.
4.  **Release:** We will release a new version of the project that includes the fix.
5.  **Disclosure:** We will publicly disclose the vulnerability after the fix has been released. We will credit you for the discovery (unless you prefer to remain anonymous).

**We kindly request that you refrain from publicly disclosing the vulnerability until we have had sufficient time to address it and release a fix.**

## Security Best Practices

This project uses Go (Golang) for the backend and JavaScript for the frontend. We strive to follow security best practices for both languages:

**Go (Golang):**

- **Dependency Management:** We use Go modules to manage dependencies and keep them up-to-date. Dependabot is used for automated dependency updates.
- **Input Validation:** All user inputs are carefully validated and sanitized to prevent common vulnerabilities like Cross-Site Scripting (XSS), SQL injection, and command injection.
- **Secure Coding Practices:** We adhere to secure coding guidelines for Go, including those outlined in the [OWASP Secure Coding Practices Quick Reference Guide](https://owasp.org/www-project-secure-coding-practices-quick-reference-guide/migrated_content)
- **Error Handling:** Proper error handling is implemented to avoid leaking sensitive information.
- **Cryptography:** We use strong, well-established cryptographic libraries (e.g., Go's `crypto`
