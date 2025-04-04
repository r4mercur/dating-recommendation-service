# User Recommendation Service

A simple microservice for managing user profiles and generating user recommendations.

## Features

- Generation of fake user profiles for testing
- Elasticsearch integration for efficient user search
- Bulk import functionality with retry mechanism
- REST API for user recommendations
- Supports up to 100,000 test user profiles
- Concurrent batch processing for better performance

## Technology Stack

- Go
- Elasticsearch
- gofakeit (for test data generation)

## Prerequisites

- Go 1.16 or higher
- Running Elasticsearch instance
- Git

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd <repository-name>
```

2. Install dependencies:
```bash
go mod download
```

3. Run the application:
```bash
go run cmd/main.go
```

4. Usage of the endpoints:
```bash
curl -X POST http://localhost:8080/users/create/fake-users
```

5. To get user recommendations:
```bash
curl -X GET http://localhost:8080/users/recommendations
```