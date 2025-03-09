# Build the manager binary
FROM golang:1.23 AS builder

LABEL NAME="Vivek Singh Bhadauriya"
LABEL EMAIL="vbhadauriya@redcloudcomputing.com"

WORKDIR /workspace

# Copy the Go source code into the workspace
COPY . .

# Set up environment variables
ENV GOPRIVATE=github.com/san-data-systems/common

# Use BuildKit's secret mount to securely pass GitHub token
RUN --mount=type=secret,id=github_token \
    GITHUB_TOKEN=$(cat /run/secrets/github_token) && \
    git config --global url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"

# Download Go modules (this will fetch from the private repo as well)
RUN GO111MODULE=on go mod download

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -o project-management-api main.go

# Create a minimal image
FROM gcr.io/distroless/static:latest
WORKDIR /
COPY --from=builder /workspace/project-management-api .

# Set the entry point for the container
ENTRYPOINT ["/project-management-api"]
