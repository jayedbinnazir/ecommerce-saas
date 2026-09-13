copy all from .env.example to .env.dev

download go
https://go.dev/dl/?utm_source=chatgpt.com 

go version (check)

enter inside every service  , like ecommerce-saas/user-management>  go mod tidy (run it)


first run----> docker compose --env-file .env.dev -f deployments/compose/docker-compose.dev.yml up --build -d kafka nginx postgres mailpit minio minio-setup redis kafka-ui 

then

docker compose --env-file .env.dev -f deployments/compose/docker-compose.dev.yml up --build user-management product-service inventory-service cart-service order-service payment-service mail-service notification-service


//u can check the service are healthy 
 hit----> localhost:8081/api/v1/health
 hit----> localhost:8082/api/v1/health
 hit----> localhost:8083/api/v1/health
            continue to change port




then run the migrations in each service inside docker container
go inside cd services/product-service  or any other services
1. From vscode powershell:>  docker exec -it product-service sh
2. cd /app 
3. go run ./cmd/migration status
4. go run ./cmd/migration up


come out of container  , type   --->  exit



now , ecommerce-saas/apiTestResults/Readme.md   ,    most of the apis are their  ,
u can test
