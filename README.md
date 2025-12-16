# TUM Meldeplattform Deployment Guide

This document describes the deployment and maintenance procedures for the TUM Meldeplattform.

## System Overview

The TUM Meldeplattform runs as a containerized application using Docker Compose with the following components:

- Traefik: Reverse proxy and certificate manager
- Website: Main application container
- Watchtower: Automatic container updates

## Directory Structure

The main application directory is located at `/root/meldeplattform/` and contains:

- `docker-compose.yml`: Container orchestration configuration
- `traefik.toml`: Traefik reverse proxy configuration
- `config.yaml`: Application configuration
- `data/`: Directory containing runtime data and certificates
- `acme/`: Directory for Let's Encrypt certificates

## Configuration

### Docker Compose

The deployment uses Docker Compose with three main services:

1. **Traefik**
   - Handles SSL/TLS termination
   - Manages automatic HTTPS redirects
   - Provides reverse proxy functionality

2. **Website**
   - Main application container
   - Mounts local configuration and data volumes
   - Exposes port 8080 internally

3. **Watchtower**
   - Monitors and automatically updates containers

## Certificate Management

The platform uses certificates for authentication. The certificates are stored in the `/root/meldeplattform/data` directory:

- `key.pem`: Private key
- `cert.pem`: Public certificate

### Certificate Renewal Process

When certificates need to be renewed:

1. Backup existing certificates:
```bash
cd /root/meldeplattform/data
mv key.pem key.pem.old
mv cert.pem cert.pem.old
```

2. Restart the website container to generate new certificates:
```bash
docker restart meldeplattform_website_1
```

3. Update the certificate in DFN-AAI:
   - Access mdv.aai.dfn.de
   - Replace the old certificate (do not add as additional)
   - Current certificate validity: January 15, 2027, 6:16 PM

Note: After certificate renewal, it may take up to an hour for the login to function properly. In some cases, an additional container restart may be required.

## Maintenance

### Container Management

To check the status of containers:
```bash
docker compose ps
```

To view container logs:
```bash
docker compose logs [service_name]
```

To restart services:
```bash
docker compose restart [service_name]
```

### Updates

Watchtower automatically handles container updates. However, manual updates can be performed:
```bash
docker compose pull
docker compose up -d
```

## Troubleshooting

### Logs

Access service logs for debugging:
```bash
# Traefik logs
docker compose logs traefik

# Website logs
docker compose logs website
```

## Go Package Management

### Updating Go Packages

To update the Go packages in the project:

1. Clone the repository locally:
```bash
git clone https://github.com/tum-dev/meldeplattform.git
cd meldeplattform
```

2. Update all dependencies to their latest versions:
```bash
go get -u ./...
```

3. Clean up the go.mod file:
```bash
go mod tidy
```

4. Test the application to ensure updates haven't introduced issues:
```bash
go test ./...
```

5. Commit the changes to go.mod and go.sum:
```bash
git add go.mod go.sum
git commit -m "chore: update go dependencies"
```

6. Create a pull request with the updates

### Version Management

- Keep track of major version updates in dependencies
- Review changelog/release notes of updated packages
- Test thoroughly after significant updates

## TODOs

- Implement certificate renewal without downtime
- Add monitoring and alerting
- Develop automated backup solutions

For additional support or questions, contact the TUM IT Support team.
