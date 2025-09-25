#!/bin/bash

# CyberArk Custom Provider - Rebuild and Run Script
# This script rebuilds the Go application, Docker container, and runs it locally

set -e # Exit on any error
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$SCRIPT_DIR"

source "$SCRIPT_DIR/color-lib.sh"
if ! [[ -f "$SCRIPT_DIR/.env" ]]; then
    error ".env file not found at $SCRIPT_DIR/.env, stopping."
    error "try running env-setup.sh to create .env"
    exit 1
fi
source "$SCRIPT_DIR/.env"

# Configuration
CONTAINER_NAME="cyberark-local-test"                           # local docker container name
IMAGE_NAME="${CONTAINER_IMAGE_NAME:-cyberark-custom-provider}" # docker image name
PORT="8080"
CUSTOM_PROVIDER_DIR="$PROJECT_DIR/custom-provider"

# Check if Docker is running
check_docker() {
    info "Checking Docker status..."
    if ! docker info >/dev/null 2>&1; then
        error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
    success "Docker is running"
}

# Stop and remove existing container
cleanup_container() {
    info "Cleaning up existing container..."
    if docker ps -q -f name=$CONTAINER_NAME | grep -q .; then
        info "Stopping running container: $CONTAINER_NAME"
        docker stop $CONTAINER_NAME >/dev/null 2>&1 || true
    fi

    if docker ps -aq -f name=$CONTAINER_NAME | grep -q .; then
        info "Removing existing container: $CONTAINER_NAME"
        docker rm $CONTAINER_NAME >/dev/null 2>&1 || true
    fi
    success "Container cleanup completed"
}

# Build Go application (optional - Docker will do this)
build_go_app() {
    info "Building Go application..."
    cd "$CUSTOM_PROVIDER_DIR"

    if command -v go >/dev/null 2>&1; then
        info "Running go mod tidy..."
        go mod tidy

        info "Building Go binary..."
        go build -o main .
        success "Go application built successfully"
    else
        warning "Go not found locally, Docker will handle the build"
    fi
}

# Build Docker image
build_docker_image() {
    info "Building Docker image: $IMAGE_NAME"
    cd "$CUSTOM_PROVIDER_DIR"

    docker build --no-cache -t $IMAGE_NAME . || {
        error "Docker build failed"
        exit 1
    }

    success "Docker image built successfully"
}

# Run new container
run_container() {
    info "Running new container: $CONTAINER_NAME"

    docker run -d \
        --name $CONTAINER_NAME \
        -p $PORT:$PORT \
        -e IDTENANTURL="$CYBERARK_ID_TENANT_URL" \
        -e PAMUSER="$CYBERARK_PAM_USER" \
        -e PAMPASS="$CYBERARK_PAM_PASSWORD" \
        -e PCLOUDURL="$CYBERARK_PCLOUD_URL" \
        $IMAGE_NAME || {
        error "Failed to start container"
        exit 1
    }

    success "Container started successfully"
}

# Wait for container to be ready
wait_for_container() {
    info "Waiting for container to be ready..."

    local max_attempts=30
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if curl -s http://localhost:$PORT/health >/dev/null 2>&1; then
            success "Container is ready and responding"
            return 0
        fi

        echo -n "."
        sleep 1
        attempt=$((attempt + 1))
    done

    error "Container failed to become ready within $max_attempts seconds"
    return 1
}

# Show container status and logs
show_status() {
    info "Container status:"
    docker ps -f name=$CONTAINER_NAME

    echo ""
    info "Recent container logs:"
    docker logs $CONTAINER_NAME --tail 10

    echo ""
    info "Testing health endpoint:"
    if command -v jq >/dev/null 2>&1; then
        curl -s http://localhost:$PORT/health | jq .
    else
        curl -s http://localhost:$PORT/health
    fi
}

# Show usage information
show_usage() {
    echo ""
    success "=== CyberArk Custom Provider - Local Testing ==="
    echo ""
    echo "Container: $CONTAINER_NAME"
    echo "Image: $IMAGE_NAME"
    echo "Port: $PORT"
    echo "Health endpoint: http://localhost:$PORT/health"
    echo ""
    echo "Available endpoints for testing:"
    echo "1. Health Check:"
    echo "   curl http://localhost:$PORT/health"
    echo ""
    echo "2. Create Safe (Action endpoint):"
    echo "   curl -X POST -H 'Content-Type: application/json' \\"
    echo "     -d '{\"properties\": {\"safeName\": \"test-safe\", \"description\": \"Test description\"}}' \\"
    echo "     'http://localhost:$PORT/subscriptions/test/resourcegroups/test/providers/Microsoft.CustomProviders/resourceProviders/test/createSafe'"
    echo ""
    echo "3. Create Safe (Resource endpoint):"
    echo "   curl -X PUT -H 'Content-Type: application/json' \\"
    echo "     -d '{\"properties\": {\"safeName\": \"test-safe\", \"description\": \"Test description\"}}' \\"
    echo "     'http://localhost:$PORT/subscriptions/test/resourcegroups/test/providers/Microsoft.CustomProviders/resourceProviders/test/cyberarkSafes/my-safe'"
    echo ""
    echo "Useful commands:"
    echo "  View logs:     docker logs -f $CONTAINER_NAME"
    echo "  Stop:          docker stop $CONTAINER_NAME"
    echo "  Restart:       docker restart $CONTAINER_NAME"
    echo ""
}

# Main execution
main() {
    echo ""
    info "=== CyberArk Custom Provider Rebuild and Run Script ==="
    echo ""

    # Check prerequisites
    check_docker

    # Execute build and run steps
    cleanup_container
    build_go_app
    build_docker_image
    run_container

    # Wait and verify
    if wait_for_container; then
        show_status
        show_usage
        success "Setup completed successfully!"
    else
        error "Setup failed - container is not responding"
        exit 1
    fi
}

# Handle script arguments
case "${1:-}" in
--help | -h)
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --help, -h     Show this help message"
    echo "  --logs, -l     Show container logs and exit"
    echo "  --stop, -s     Stop the container and exit"
    echo "  --status       Show container status and exit"
    echo ""
    echo "Default: Rebuild and run the container"
    exit 0
    ;;
--logs | -l)
    docker logs -f $CONTAINER_NAME
    exit 0
    ;;
--stop | -s)
    info "Stopping container: $CONTAINER_NAME"
    docker stop $CONTAINER_NAME
    success "Container stopped"
    exit 0
    ;;
--status)
    show_status
    exit 0
    ;;
"")
    # No arguments - run main process
    main
    ;;
*)
    error "Unknown option: $1"
    echo "Use --help for usage information"
    exit 1
    ;;
esac
