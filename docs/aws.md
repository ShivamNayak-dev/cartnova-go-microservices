# AWS Deployment Plan

CartNova is intentionally designed so local development works with Docker Compose while AWS can replace infrastructure components with managed services.

## Recommended mapping

| Local component | AWS direction |
|---|---|
| Go services | ECS on Fargate or EC2 |
| PostgreSQL | Amazon RDS for PostgreSQL |
| Redis | Amazon ElastiCache for Redis |
| Kafka | Amazon MSK |
| MongoDB | MongoDB Atlas or a managed MongoDB deployment |
| Secrets | AWS Secrets Manager |
| Container images | Amazon ECR |
| HTTPS entry point | Application Load Balancer |

For a learning project, ECS/Fargate is a reasonable next step. An EC2-only deployment is also acceptable if cost and simplicity are the priority.

## Network layout

Keep databases and Kafka private. Only the load balancer should be publicly reachable.

```text
Internet
   |
   v
ALB / HTTPS
   |
   v
API Gateway
   |
   +---- ECS services
            |
            +---- RDS PostgreSQL
            +---- ElastiCache Redis
            +---- MSK Kafka
            +---- MongoDB Atlas
```

## Security rules

- Never commit database passwords or JWT secrets.
- Store production secrets in Secrets Manager.
- Use private subnets for databases.
- Restrict security groups to required service-to-service traffic.
- Use HTTPS at the public boundary.
- Rotate JWT/database credentials.
- Remove the seeded demo admin account from production.

## Deployment sequence

1. Create the VPC and subnets.
2. Create RDS databases or separate logical databases for service ownership.
3. Create Redis.
4. Create Kafka/MSK.
5. Provision MongoDB/Atlas.
6. Build Docker images.
7. Push images to ECR.
8. Deploy services.
9. Configure environment variables/secrets.
10. Deploy the API Gateway behind the ALB.
11. Verify health endpoints.
12. Run API smoke tests.

The exact AWS resource choices can be adjusted when deployment begins; the application code should not be rewritten just to match a particular AWS service.
