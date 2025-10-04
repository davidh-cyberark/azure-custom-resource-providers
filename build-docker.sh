#!/bin/bash

# Docker build and push script for CyberArk Custom Provider
# This script builds the Docker image and optionally pushes it to Azure Container Registry

set -e # Exit on any error

set -a
source color-lib.sh
source .env

# Configuration
IMAGE_NAME="${CONTAINER_IMAGE_NAME:-cyberark-custom-provider}"
DOCKERFILE_PATH="./custom-provider"
BUILD_CONTEXT="./custom-provider"
PUSH_TO_ACR=false

if [[ -f "${BUILD_CONTEXT}/VERSION" ]]; then
    BUILD_VERSION="v$(cat ${BUILD_CONTEXT}/VERSION)"
fi
IMAGE_TAG="${BUILD_VERSION:-latest}"
set +a

# Function to check if Docker is running
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        error "Docker is not running or not accessible. Please start Docker and try again."
        exit 1
    fi
}

# Function to build the Docker image
build_image() {
    local image_name="$1"
    local image_tag="$2"
    local full_image_name="${image_name}:${image_tag}"

    info "Building Docker image: ${full_image_name}"
    info "Build context: ${BUILD_CONTEXT}"
    info "Dockerfile path: ${DOCKERFILE_PATH}/Dockerfile"

    if docker build -t "${full_image_name}" -f "${DOCKERFILE_PATH}/Dockerfile" "${BUILD_CONTEXT}"; then
        success "Docker image built successfully: ${full_image_name}"
        return 0
    fi

    error "Failed to build Docker image"
    return 1
}

# Function to push image to ACR
push_to_acr() {
    local local_image="$1"
    local acr_image="$2"

    if [[ -z "$ACR_NAME" || -z "$ACR_LOGIN_SERVER" ]]; then
        error "ACR_NAME and ACR_LOGIN_SERVER must be set in environment variables"
        return 1
    fi

    info "Tagging image for ACR: ${acr_image}"
    if ! docker tag "${local_image}" "${acr_image}"; then
        error "Failed to tag image for ACR"
        return 1
    fi

    info "Logging into Azure Container Registry: ${ACR_NAME}"
    if ! az acr login --name "${ACR_NAME}"; then
        error "Failed to login to ACR"
        return 1
    fi

    info "Pushing image to ACR: ${acr_image}"
    if docker push "${acr_image}"; then
        success "Image pushed successfully to ACR: ${acr_image}"
        return 0
    fi

    error "Failed to push image to ACR"
    return 1
}

# Function to show image info
show_image_info() {
    local image_name="$1"

    info "Docker image information:"
    docker images "${image_name}" --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.Size}}\t{{.CreatedAt}}"
}

# Function to test the image (optional)
test_image() {
    local full_image_name="$1"
    local test_name="${IMAGE_NAME}-test"

    info "Testing if the image can start..."
    info "Image name: ${full_image_name}"

    if docker run --rm -d --name "${test_name}" -p 8080:8080 \
        -e IDTENANTURL="$CYBERARK_ID_TENANT_URL" \
        -e PAMUSER="$CYBERARK_PAM_USER" \
        -e PAMPASS="$CYBERARK_PAM_PASSWORD" \
        -e PCLOUDURL="$CYBERARK_PCLOUD_URL" \
        "${full_image_name}" >/dev/null; then
        sleep 2
        if docker ps | grep -q "${test_name}"; then
            success "Image starts successfully"
            docker stop "${test_name}" >/dev/null
        else
            warning "Image may have issues starting"
        fi
    else
        warning "Could not test image startup"
    fi
}

# Main execution
main() {
    info "Starting Docker build process for CyberArk Custom Provider"
    info "Using image name: ${IMAGE_NAME}"
    info "Using version tag: ${IMAGE_TAG}"

    # Check if we're in the right directory
    if [[ ! -d "${BUILD_CONTEXT}" ]]; then
        error "Build context directory '${BUILD_CONTEXT}' not found. Please run this script from the project root directory."
        exit 1
    fi

    if [[ ! -f "${DOCKERFILE_PATH}/Dockerfile" ]]; then
        error "Dockerfile not found at '${DOCKERFILE_PATH}/Dockerfile'"
        exit 1
    fi

    # Check Docker
    check_docker

    # Define image names
    local local_image="${IMAGE_NAME}:${IMAGE_TAG}"
    local acr_image=""

    if [[ "$PUSH_TO_ACR" == "true" && -n "$ACR_LOGIN_SERVER" ]]; then
        acr_image="${ACR_LOGIN_SERVER}/${IMAGE_NAME}:${IMAGE_TAG}"
    fi

    # Build the image
    if ! build_image "${IMAGE_NAME}" "${IMAGE_TAG}"; then
        error "Build process failed!"
        return 1
    fi

    show_image_info "${IMAGE_NAME}"

    # Push to ACR if requested
    if [[ "$PUSH_TO_ACR" == "true" ]]; then
        if ! push_to_acr "${local_image}" "${acr_image}"; then
            error "Build succeeded but push to ACR failed!"
            return 1
        fi
        success "Image successfully built and pushed to ACR!"
        info "ACR Image: ${acr_image}"
    fi

    if [ "$SKIP_TEST" != "true" ]; then
        # Ask if user wants to test the image (only for local builds)
        echo
        read -p "Do you want to test if the image can start? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            test_image "${local_image}"
        fi
    fi

    success "Build process completed successfully!"
    echo
    info "You can now run the container with:"
    echo "  docker run -d -p 8080:8080 ${local_image}"

    if [[ -n "$ACR_LOGIN_SERVER" ]]; then
        echo
        info "To push to ACR later, run:"
        echo "  $0 --push"
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
    -t | --tag)
        IMAGE_TAG="$2"
        shift 2
        ;;
    -n | --name)
        IMAGE_NAME="$2"
        shift 2
        ;;
    -p | --push)
        PUSH_TO_ACR=true
        shift
        ;;
    --no-test)
        SKIP_TEST=true
        shift
        ;;
    -h | --help)
        echo "Usage: $0 [OPTIONS]"
        echo "Build Docker image for CyberArk Custom Provider"
        echo ""
        echo "Options:"
        echo "  -t, --tag TAG       Docker image tag (default: auto-detected from VERSION file)"
        echo "  -n, --name NAME     Docker image name (default: extracted from CONTAINER_IMAGE env var)"
        echo "  -p, --push          Push to Azure Container Registry after building"
        echo "  --no-test           Skip the startup test"
        echo "  -h, --help          Show this help message"
        echo ""
        echo "Environment variables:"
        echo "  CONTAINER_IMAGE     Used to extract image name (e.g., 'myimage:tag' -> 'myimage')"
        echo "  ACR_NAME            Azure Container Registry name (required for --push)"
        echo "  ACR_LOGIN_SERVER    ACR login server URL (required for --push)"
        exit 0
        ;;
    *)
        error "Unknown option: $1"
        echo "Use -h or --help for usage information"
        exit 1
        ;;
    esac
done

# Run main function
main
