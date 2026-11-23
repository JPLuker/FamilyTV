# Foundation verification

GitHub Actions run [45240642565](https://github.com/JPLuker/FamilyTV/actions/runs/45240642565) passed the backend race tests, Go vet and production web build for commit a34fd7540e6bf332b53828c136d1603296168076.

Run `bash scripts/compose-smoke.sh` with Docker Compose 2.24.4 or later, Bash, curl and jq. It builds the production image, creates a parent and a profile, recreates the container, checks the stored profile and signs out. The script creates and removes its own volume. It uses an ephemeral loopback port.

Device accessibility, backup recovery and internet deployment still require separate verification.
