download-cards:
    go run ./cmd/rb download-cards -out cards/

collect set="":
    go run ./cmd/rb collect -catalog-file cards/cards.json -collection-file collection/collection.json {{set}}

validate set="":
    go run ./cmd/rb validate -collection-file collection/collection.json {{set}}

collection-stats:
    go run ./cmd/rb collection-stats -catalog-file cards/cards.json -collection-file collection/collection.json

gen-anki:
    go run ./cmd/rb anki-gen -catalog-file cards/cards.json -image-dir cards/images -out anki/
