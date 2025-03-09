## Deploying with Podman Compose

To deploy the application using **Podman Compose**, follow these steps:

### Prerequisites
Ensure you have the following installed:
- [Podman](https://podman.io/getting-started/installation)
- [Podman Compose](https://github.com/containers/podman-compose)
- A valid `.env` file in the project root directory

### Deployment Steps

**1. Navigate to the project directory**
   ```bash
   cd /path/to/project
   PWD=$(pwd)
   ```
**2. Stop any running containers (if applicable)**
```bash
podman-compose -f ${PWD}/deploy/docker-compose/docker-compose.yml down
```

**3. Start the containers using Podman Compose**
```bash
podman-compose -f ${PWD}/deploy/docker-compose/docker-compose.yml --env-file ${PWD}/.env up -d
```

**4. Verify running containers**
```bash
podman ps
```

**5. Stopping the Deployment**
```bash
podman-compose -f ${PWD}/deploy/docker-compose/docker-compose.yml down
```