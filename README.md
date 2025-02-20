# File Storage Service

Микросервис для хранения файлов с использованием Minio и MongoDB.

## Требования

-   Go 1.19+
-   Minio server
-   MongoDB

## Установка

1. Склонировать репозиторий
2. Настроить .env файл
3. Запустить:

```bash
go run main.go
```

Вот полный набор команд для запуска проекта:

1. Запуск MinIO

Windows

```bash
docker run -d --name minio  -p 9000:9000 -p 9001:9001 -e "MINIO_ROOT_USER=myadmin"  -e "MINIO_ROOT_PASSWORD=mysecretpassword" -v C:\minio-data:/data minio/minio server /data --console-address ":9001"
```

macOS

```bash
docker run -d --name minio -p 9000:9000 -p 9001:9001 -e "MINIO_ROOT_USER=myadmin" -e "MINIO_ROOT_PASSWORD=mysecretpassword" -v ~/minio-data:/data minio/minio server /data --console-address ":9001"
```

2. Настройка MinIO

# Создать бакет

```bash
docker exec -it minio mc alias set myminio http://localhost:9000 myadmin mysecretpassword
docker exec -it minio mc mb myminio/user-uploads
docker exec -it minio mc anonymous set public myminio/user-uploads
```

3. Запуск MongoDB

Windows

```bash
docker run -d --name mongo -p 27017:27017 -e MONGO_INITDB_ROOT_USERNAME=admin -e MONGO_INITDB_ROOT_PASSWORD=secret -v C:\mongo-data:/data/db mongo:latest
```

macOS

```bash
docker run -d --name mongo -p 27017:27017 -e MONGO_INITDB_ROOT_USERNAME=admin -e MONGO_INITDB_ROOT_PASSWORD=secret -v ~/mongo-data:/data/db mongo:latest
```

4. Настройка окружения (.env файл)

Создайте в корне проекта файл .env:

```bash
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=myadmin
MINIO_SECRET_KEY=mysecretpassword
MINIO_SSL=false

MONGO_URI=mongodb://admin:secret@localhost:27017/file_storage?authSource=admin
MONGO_DATABASE=file_storage

SERVER_PORT=:8080
API_KEY=secure-api-key-123
```

5. Установка зависимостей

```bash
go mod tidy
```

6. Generate Swager

```bash
$(go env GOPATH)/bin/swag init
$(go env GOPATH)/bin/swag init cmd/server/main.go
```

7. Запуск сервера

```bash
go run cmd/server/main.go
```

8. Проверка работы (примеры запросов)

Загрузка файла:

```bash
curl -X POST -H "Authorization: secure-api-key-123" -F "file=@C:\Path\To\File.jpg" http://localhost:8080/api/v1/upload
```

Получение метаданных:

```bash
curl -H "Authorization: secure-api-key-123" http://localhost:8080/api/v1/files/ваш-id-файла
```

Удаление файла:

```bash
curl -X DELETE -H "Authorization: secure-api-key-123" http://localhost:8080/api/v1/files/ваш-id-файла
```

# Если возникают проблемы:

## Проверьте статус контейнеров:

```bash
docker ps -a
```

## Просмотрите логи:

```bash
docker logs minio
docker logs mongo
```

## Для остановки всех сервисов:

```bash
docker stop minio mongo
docker rm minio mongo
```

9. Проверка через веб-интерфейсы

## MinIO Console: `http://localhost:9001`

Логин: `myadmin`
Пароль: `mysecretpassword`

### Swagger UI: `http://localhost:8080/swagger/index.html`

Available authorizations
ApiKeyAuth (apiKey)
Value:`secure-api-key-123`

## MongoDB Compass:

Hostname: `localhost`
Port: `27017`
Username: `admin`
Password: `secret`
Authentication Database: `admin`
