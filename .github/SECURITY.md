# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest  | Yes       |

We recommend always running the latest release. Older releases do not receive security backports.

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Email **security@mittolabs.com** with:

- A description of the vulnerability and its potential impact
- Steps to reproduce or a proof-of-concept
- Any suggested mitigations you have identified

You will receive an acknowledgement within **48 hours**. We aim to release a patch within **14 days** of a confirmed report and will credit you in the release notes (unless you prefer to remain anonymous).

## Scope

In scope:
- Authentication and session management
- API key hashing and storage
- Multi-tenant data isolation (project schemas)
- Credential encryption
- SQL injection and other injection attacks
- Cross-site scripting (XSS) in the admin console
- Insecure default configurations

Out of scope:
- Vulnerabilities in third-party dependencies that have no upstream fix
- Issues that require physical access to the host
- Social engineering
