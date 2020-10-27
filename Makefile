pg:
	docker run --rm --name pg -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:13-alpine

kill_pg:
	docker kill pg

createdb:
	docker exec -it pg createdb --username=root --owner=root cells >/dev/null; true

dropdb:
	docker exec -it pg dropdb cells

migrateup: createdb
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/cells?sslmode=disable" -verbose up

migratedown:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/cells?sslmode=disable" -verbose down

server:
	go run main.go

test:
	go test

.PHONY: postgres createdb dropdb migrateup migratedown server test kill_pg