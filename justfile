download-cards:
    go run ./cmd/rb download-cards -out catalog/

collection:
    go run ./cmd/rb collection -catalog-file catalog/cards.json -collection-file collection/collection.json

anki-gen:
    go run ./cmd/rb anki-gen -catalog-file catalog/cards.json -image-dir catalog/images -out anki/
