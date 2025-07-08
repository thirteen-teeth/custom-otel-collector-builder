#!/bin/bash

# Version management script for custom-otel-collector-builder
# This script provides utilities to keep IMAGE_TAG and git tags in sync

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get current IMAGE_TAG from Makefile
get_image_tag() {
    grep "^IMAGE_TAG=" Makefile | cut -d'=' -f2
}

# Get latest git tag
get_latest_git_tag() {
    git describe --tags --abbrev=0 2>/dev/null || echo "none"
}

# Check if working directory is clean
is_working_directory_clean() {
    git diff --quiet && git diff --cached --quiet
}

# Display current status
status() {
    echo -e "${BLUE}=== Version Status ===${NC}"
    local image_tag=$(get_image_tag)
    local git_tag=$(get_latest_git_tag)
    
    echo -e "IMAGE_TAG in Makefile: ${YELLOW}$image_tag${NC}"
    echo -e "Latest git tag:        ${YELLOW}$git_tag${NC}"
    
    if [ "$git_tag" = "none" ]; then
        echo -e "Status: ${RED}No git tags found${NC}"
        return 1
    elif [ "$image_tag" = "$git_tag" ]; then
        echo -e "Status: ${GREEN}✅ In sync${NC}"
        return 0
    else
        echo -e "Status: ${RED}❌ Out of sync${NC}"
        return 1
    fi
}

# Validate version format
validate_version() {
    local version=$1
    if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo -e "${RED}Error: Invalid version format. Expected format: x.y.z (e.g., 1.0.0)${NC}"
        return 1
    fi
    return 0
}

# Set IMAGE_TAG to a specific version
set_version() {
    local new_version=$1
    if [ -z "$new_version" ]; then
        echo -e "${RED}Error: Please provide a version number${NC}"
        echo "Usage: $0 set 1.0.0"
        return 1
    fi
    
    validate_version "$new_version" || return 1
    
    echo -e "${BLUE}Setting IMAGE_TAG to $new_version${NC}"
    sed -i "s/^IMAGE_TAG=.*/IMAGE_TAG=$new_version/" Makefile
    echo -e "${GREEN}✅ IMAGE_TAG updated to $new_version${NC}"
}

# Show help
show_help() {
    echo "Version management script for custom-otel-collector-builder"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  status                  Show current version status"
    echo "  set <version>          Set IMAGE_TAG to specific version (e.g., 1.0.0)"
    echo "  sync-from-git          Set IMAGE_TAG to match latest git tag"
    echo "  check                  Check if IMAGE_TAG matches latest git tag (exit 0 if sync)"
    echo "  next-patch             Show what the next patch version would be"
    echo "  next-minor             Show what the next minor version would be"
    echo "  next-major             Show what the next major version would be"
    echo "  help                   Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 status              # Show current version status"
    echo "  $0 set 1.2.3           # Set IMAGE_TAG to 1.2.3"
    echo "  $0 sync-from-git       # Sync IMAGE_TAG with latest git tag"
    echo "  $0 check               # Check if versions are in sync"
    echo ""
    echo "Note: Use the Makefile targets for releasing:"
    echo "  make increment-patch   # Increment patch version"
    echo "  make increment-minor   # Increment minor version"
    echo "  make increment-major   # Increment major version"
    echo "  make quick-release     # Increment patch and release"
    echo "  make release           # Create release with current IMAGE_TAG"
}

# Show next version
show_next_version() {
    local type=$1
    local current_tag=$(get_image_tag)
    
    validate_version "$current_tag" || return 1
    
    local major=$(echo "$current_tag" | cut -d. -f1)
    local minor=$(echo "$current_tag" | cut -d. -f2)
    local patch=$(echo "$current_tag" | cut -d. -f3)
    
    case $type in
        patch)
            echo "$major.$minor.$((patch + 1))"
            ;;
        minor)
            echo "$major.$((minor + 1)).0"
            ;;
        major)
            echo "$((major + 1)).0.0"
            ;;
        *)
            echo -e "${RED}Error: Unknown version type: $type${NC}"
            return 1
            ;;
    esac
}

# Sync IMAGE_TAG from git
sync_from_git() {
    local git_tag=$(get_latest_git_tag)
    
    if [ "$git_tag" = "none" ]; then
        echo -e "${RED}Error: No git tags found. Cannot sync IMAGE_TAG.${NC}"
        return 1
    fi
    
    echo -e "${BLUE}Syncing IMAGE_TAG with latest git tag: $git_tag${NC}"
    sed -i "s/^IMAGE_TAG=.*/IMAGE_TAG=$git_tag/" Makefile
    echo -e "${GREEN}✅ IMAGE_TAG updated to $git_tag${NC}"
}

# Check if versions are in sync
check_sync() {
    local image_tag=$(get_image_tag)
    local git_tag=$(get_latest_git_tag)
    
    if [ "$git_tag" = "none" ]; then
        echo -e "${RED}❌ No git tags found. Current IMAGE_TAG: $image_tag${NC}"
        return 1
    elif [ "$image_tag" != "$git_tag" ]; then
        echo -e "${RED}❌ IMAGE_TAG ($image_tag) does not match latest git tag ($git_tag)${NC}"
        return 1
    else
        echo -e "${GREEN}✅ IMAGE_TAG ($image_tag) matches latest git tag ($git_tag)${NC}"
        return 0
    fi
}

# Main command handling
case "${1:-status}" in
    status)
        status
        ;;
    set)
        set_version "$2"
        ;;
    sync-from-git)
        sync_from_git
        ;;
    check)
        check_sync
        ;;
    next-patch)
        show_next_version patch
        ;;
    next-minor)
        show_next_version minor
        ;;
    next-major)
        show_next_version major
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo -e "${RED}Error: Unknown command: $1${NC}"
        echo "Use '$0 help' for usage information."
        exit 1
        ;;
esac
