download-cards:
    go run ./cmd/rb download-cards -out cards/

collection:
    go run ./cmd/rb collection -cards cards/ -file collection.json
