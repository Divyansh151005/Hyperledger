#!/bin/bash

################################################################################
# Network Management Script
# This script manages the lifecycle of the Hyperledger Fabric network
################################################################################

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NETWORK_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DOCKER_DIR="${NETWORK_DIR}/docker"

# Docker compose files
CA_COMPOSE="${DOCKER_DIR}/docker-compose-ca.yaml"
ORDERER_COMPOSE="${DOCKER_DIR}/docker-compose-orderer.yaml"
PEER_COMPOSE="${DOCKER_DIR}/docker-compose-peer.yaml"

# Function to print usage
usage() {
    echo -e "${YELLOW}Usage: $0 {up|down|restart|clean|logs}${NC}"
    echo -e "${YELLOW}  up      - Start the network${NC}"
    echo -e "${YELLOW}  down    - Stop the network${NC}"
    echo -e "${YELLOW}  restart - Restart the network${NC}"
    echo -e "${YELLOW}  clean   - Stop and remove all containers, volumes, and networks${NC}"
    echo -e "${YELLOW}  logs    - Show logs from all containers${NC}"
    exit 1
}

# Function to check prerequisites
check_prerequisites() {
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}Error: Docker is not installed${NC}"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        echo -e "${RED}Error: Docker Compose is not installed${NC}"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        echo -e "${RED}Error: Docker daemon is not running${NC}"
        exit 1
    fi
}

# Function to start network
start_network() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Starting Medical Network${NC}"
    echo -e "${GREEN}========================================${NC}"
    
    check_prerequisites
    
    # Start CAs first
    echo -e "${BLUE}Starting Certificate Authorities...${NC}"
    docker-compose -f "${CA_COMPOSE}" up -d
    
    # Wait for CAs to be ready
    echo -e "${YELLOW}Waiting for CAs to be ready...${NC}"
    sleep 5
    
    # Start orderers
    echo -e "${BLUE}Starting Orderer nodes...${NC}"
    docker-compose -f "${ORDERER_COMPOSE}" up -d
    
    # Wait for orderers to be ready
    echo -e "${YELLOW}Waiting for Orderers to be ready...${NC}"
    sleep 10
    
    # Start peers
    echo -e "${BLUE}Starting Peer nodes...${NC}"
    docker-compose -f "${PEER_COMPOSE}" up -d
    
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Network started successfully!${NC}"
    echo -e "${GREEN}========================================${NC}"
    
    # Show running containers
    echo -e "\n${YELLOW}Running containers:${NC}"
    docker ps --filter "network=medical-network" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
}

# Function to stop network
stop_network() {
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}Stopping Medical Network${NC}"
    echo -e "${YELLOW}========================================${NC}"
    
    docker-compose -f "${PEER_COMPOSE}" down
    docker-compose -f "${ORDERER_COMPOSE}" down
    docker-compose -f "${CA_COMPOSE}" down
    
    echo -e "${GREEN}Network stopped successfully!${NC}"
}

# Function to restart network
restart_network() {
    echo -e "${YELLOW}Restarting network...${NC}"
    stop_network
    sleep 2
    start_network
}

# Function to clean network
clean_network() {
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}Cleaning Medical Network${NC}"
    echo -e "${RED}========================================${NC}"
    
    echo -e "${YELLOW}This will remove all containers, volumes, and networks.${NC}"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}Clean cancelled${NC}"
        exit 0
    fi
    
    # Stop and remove containers
    docker-compose -f "${PEER_COMPOSE}" down -v
    docker-compose -f "${ORDERER_COMPOSE}" down -v
    docker-compose -f "${CA_COMPOSE}" down -v
    
    # Remove network if it exists
    docker network rm medical-network 2>/dev/null || true
    
    echo -e "${GREEN}Network cleaned successfully!${NC}"
}

# Function to show logs
show_logs() {
    echo -e "${BLUE}Showing network logs (Ctrl+C to exit)...${NC}"
    docker-compose -f "${CA_COMPOSE}" -f "${ORDERER_COMPOSE}" -f "${PEER_COMPOSE}" logs -f
}

# Main script logic
case "$1" in
    up)
        start_network
        ;;
    down)
        stop_network
        ;;
    restart)
        restart_network
        ;;
    clean)
        clean_network
        ;;
    logs)
        show_logs
        ;;
    *)
        usage
        ;;
esac
