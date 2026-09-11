First i create all databases for microservices by platform docker compose
In Infra/Postgres/init/01-create-databases.sql
CREATE DATABASE user_management_db;
CREATE DATABASE product_db;
CREATE DATABASE inventory_db;
CREATE DATABASE cart_db;
CREATE DATABASE order_db;
CREATE DATABASE payment_db;
CREATE DATABASE notification_db;
CREATE DATABASE mail_db;

now in docker-compose we create postgres server container and mount the
volume  infra/postgres/init/01-


postgres-data is empty
        ↓
PostgreSQL initializes
        ↓
runs SQL files from /docker-entrypoint-initdb.d
        ↓
database created

services:
  postgres:
    image: postgres:17-alpine
    container_name: ecommerce-postgres
    restart: unless-stopped

    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: postgres

    ports:
      - "5432:5432"

    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ../../infrastructure/postgres/init:/docker-entrypoint-initdb.d:ro

    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 10

    networks:
      - ecommerce-network

docker compose -f deployments/compose/docker-compose.dev.yml up -d
docker logs ecommerce-postgres
docker logs -f ecommerce-postgres
docker exec -it ecommerce-postgres psql -U postgres -c "\l"  //list all
docker compose -f deployments/compose/docker-compose.dev.yml down -v  
docker compose -f deployments/compose/docker-compose.dev.yml config --services
docker volume ls