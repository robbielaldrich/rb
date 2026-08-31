download-cards:
    go run ./cmd/rb download-cards -out cards/

# filters are set labels and domains, in any order: `just collect VEN chaos`
collect *filters:
    go run ./cmd/rb collect -catalog-file cards/cards.json -collection-file collection/collection.json {{filters}}

validate set="":
    go run ./cmd/rb validate -collection-file collection/collection.json {{set}}

collection-stats:
    go run ./cmd/rb collection-stats -catalog-file cards/cards.json -collection-file collection/collection.json

surplus:
    go run ./cmd/rb surplus -catalog-file cards/cards.json -collection-file collection/collection.json

# filters are set labels and domains, in any order: `just missing VEN chaos`
missing *filters:
    go run ./cmd/rb missing -catalog-file cards/cards.json -collection-file collection/collection.json {{filters}}

gen-anki:
    go run ./cmd/rb gen-anki -catalog-file cards/cards.json -image-dir cards/images -out anki/

add-decks:
    go run ./cmd/rb add-decks -catalog-file cards/cards.json -decks-file decks/decks.json

match-decks *flags:
    go run ./cmd/rb match-decks -catalog-file cards/cards.json -collection-file collection/collection.json -decks-file decks/decks.json -out decks/match-decks-result.txt {{flags}}
