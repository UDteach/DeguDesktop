# Windows Release Trust

This note tracks the Windows AV, SmartScreen, and updater hardening steps for Degu Desktop.

## Implemented In Repo

- The release workflow publishes `SHA256SUMS.txt` for release ZIPs and packaged `DeguDesktop.exe` files.
- Each Windows release ZIP includes `SECURITY.txt` with the expected EXE hash, expected behavior, source/release URLs, and Microsoft/McAfee false-positive submission guidance.
- The app updater verifies GitHub release asset size and `sha256:` digest when GitHub provides it.
- The runtime updater no longer writes or launches a PowerShell script and no longer uses `ExecutionPolicy Bypass`; it copies the current EXE as a temporary helper and applies the update through the app's own constrained updater mode.
- The updater helper accepts only a source `DeguDesktop.exe` inside the app-owned update temp directory and a target `DeguDesktop.exe` outside that temp directory.
- The release workflow embeds Win32 version metadata, product metadata, a Windows 10+ manifest, and an app icon into `DeguDesktop.exe`.
- Release builds no longer use `-s -w`, leaving richer PE/debug metadata in the Go binary.
- Optional Authenticode signing is wired into the release workflow through Microsoft Azure Artifact Signing or fallback `.pfx` GitHub Secrets.

## Release Signing

### Preferred: Azure Artifact Signing

Configure these GitHub Secrets when Microsoft Azure Artifact Signing / Trusted Signing is available:

- `AZURE_CLIENT_ID`
- `AZURE_TENANT_ID`
- `AZURE_SUBSCRIPTION_ID`
- `AZURE_ARTIFACT_SIGNING_ENDPOINT`
- `AZURE_ARTIFACT_SIGNING_ACCOUNT_NAME`
- `AZURE_ARTIFACT_SIGNING_CERTIFICATE_PROFILE_NAME`

The release workflow logs in with GitHub OIDC, signs both Windows EXEs with `azure/artifact-signing-action`, and then packages the signed EXEs into ZIPs.

### Fallback: PFX Signing

Use this only for a certificate/key that is legitimately exportable:

- `WINDOWS_CERTIFICATE_BASE64`: base64-encoded `.pfx`
- `WINDOWS_CERTIFICATE_PASSWORD`: `.pfx` password

If signing secrets are absent, the workflow builds and publishes checksums, but the EXE remains unsigned. Unsigned new binaries may still trigger reputation warnings.

## Operational Checklist

1. Build and publish stable ZIP names:
   - `DeguDesktop-windows-amd64.zip`
   - `DeguDesktop-windows-386.zip`
2. Include `README.md` and `SECURITY.txt` inside each ZIP.
3. Publish `SHA256SUMS.txt` with ZIP and inner EXE hashes.
4. Run Microsoft Defender custom scan on the local built EXE before release when possible.
5. If blocked, submit the exact ZIP/EXE to Microsoft and McAfee with the release URL and hashes.

## References

- Microsoft SmartScreen reputation: https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/smartscreen-reputation
- Microsoft code-signing options: https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/code-signing-options
- Azure Artifact Signing integrations: https://learn.microsoft.com/en-us/azure/artifact-signing/how-to-signing-integrations
- Microsoft Security Intelligence file submission: https://www.microsoft.com/en-us/wdsi/filesubmission
- McAfee Dispute Detection & Allowlisting: https://www.mcafee.com/en-us/consumer-support/dispute-detection-allowlisting.html
- GitHub release asset digest field: https://docs.github.com/en/rest/releases/assets
