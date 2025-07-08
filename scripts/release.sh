#!/bin/bash

# Zabbix Docker Swarm Plugin Release Script
# This script helps automate the release process

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're in the right directory
if [ ! -f "src/main.go" ] || [ ! -f "README.md" ]; then
    print_error "Please run this script from the project root directory"
    exit 1
fi

# Check if git is clean
if [ -n "$(git status --porcelain)" ]; then
    print_error "Working directory is not clean. Please commit or stash changes first."
    git status --short
    exit 1
fi

# Get current version from main.go
CURRENT_MAJOR=$(grep "PLUGIN_VERSION_MAJOR = " src/main.go | awk '{print $3}')
CURRENT_MINOR=$(grep "PLUGIN_VERSION_MINOR = " src/main.go | awk '{print $3}')
CURRENT_PATCH=$(grep "PLUGIN_VERSION_PATCH = " src/main.go | awk '{print $3}')
CURRENT_RC=$(grep "PLUGIN_VERSION_RC" src/main.go | awk -F'"' '{print $2}')

CURRENT_VERSION="v${CURRENT_MAJOR}.${CURRENT_MINOR}.${CURRENT_PATCH}"
if [ -n "$CURRENT_RC" ]; then
    CURRENT_VERSION="${CURRENT_VERSION}-${CURRENT_RC}"
fi

print_info "Current version: $CURRENT_VERSION"

# Ask for new version
echo ""
echo "Enter new version information:"
read -p "Major version ($CURRENT_MAJOR): " NEW_MAJOR
read -p "Minor version ($CURRENT_MINOR): " NEW_MINOR
read -p "Patch version ($CURRENT_PATCH): " NEW_PATCH
read -p "Release candidate (current: '$CURRENT_RC', leave empty for stable): " NEW_RC

# Use current values if empty
NEW_MAJOR=${NEW_MAJOR:-$CURRENT_MAJOR}
NEW_MINOR=${NEW_MINOR:-$CURRENT_MINOR}
NEW_PATCH=${NEW_PATCH:-$CURRENT_PATCH}

NEW_VERSION="v${NEW_MAJOR}.${NEW_MINOR}.${NEW_PATCH}"
if [ -n "$NEW_RC" ]; then
    NEW_VERSION="${NEW_VERSION}-${NEW_RC}"
fi

print_info "New version will be: $NEW_VERSION"

# Confirm
echo ""
read -p "Proceed with release $NEW_VERSION? (y/N): " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_warning "Release cancelled"
    exit 0
fi

# Update version in main.go
print_info "Updating version in src/main.go..."
sed -i.bak \
    -e "s/PLUGIN_VERSION_MAJOR = .*/PLUGIN_VERSION_MAJOR = $NEW_MAJOR/" \
    -e "s/PLUGIN_VERSION_MINOR = .*/PLUGIN_VERSION_MINOR = $NEW_MINOR/" \
    -e "s/PLUGIN_VERSION_PATCH = .*/PLUGIN_VERSION_PATCH = $NEW_PATCH/" \
    -e "s/PLUGIN_VERSION_RC.*= .*/PLUGIN_VERSION_RC    = \"$NEW_RC\"/" \
    src/main.go

rm src/main.go.bak

# Verify the changes
print_info "Updated version information:"
grep "PLUGIN_VERSION" src/main.go

# Build and test
print_info "Building plugin to verify changes..."
cd src
make clean
make build-all
cd ..

print_success "Build successful!"

# Commit changes
print_info "Committing version changes..."
git add src/main.go
git commit -m "Release $NEW_VERSION"

# Create and push tag
print_info "Creating git tag $NEW_VERSION..."
git tag "$NEW_VERSION"

print_success "Local preparation complete!"
print_info "To complete the release, push the changes and tag:"
echo ""
echo "  git push origin main"
echo "  git push origin $NEW_VERSION"
echo ""
print_info "After pushing the tag, GitHub Actions will automatically:"
echo "  - Build binaries for x86_64 and ARM64"
echo "  - Create a GitHub release"
echo "  - Upload binaries and packages"
echo "  - Generate release notes"
echo ""

# Ask if user wants to push automatically
read -p "Push changes and tag now? (y/N): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_info "Pushing to origin..."
    git push origin main
    git push origin "$NEW_VERSION"
    print_success "Release $NEW_VERSION has been pushed!"
    print_info "Check GitHub Actions for build progress: https://github.com/$(git remote get-url origin | sed 's/.*github.com[:/]\([^.]*\).*/\1/')/actions"
else
    print_warning "Remember to push manually:"
    echo "  git push origin main"
    echo "  git push origin $NEW_VERSION"
fi 