### Prerequisites
- Go 1.23+
- Docker

### Local Execution
```bash
git clone https://github.com/your-repo/load-balancer.git
cd load-balancer
go run ./cmd/app --config=./config.yaml
```

### Docker Deployment
```bash
docker-compose up --build
```

### API
| Method | Endpoint      | Description                   |
|--------|---------------|------------------------------|
| POST   | /clients      | Add client rate limits |
| DELETE | /clients/{id} | Remove client configuration   |
