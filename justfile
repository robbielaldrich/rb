# download card list
download-cards:
    go run ./cmd/rb download-cards -out cards/

collect *filters:
    # filters are set labels and domains, in any order: `just collect VEN chaos`
    go run ./cmd/rb collect -catalog-file cards/cards.json -collection-file collection/collection.json {{filters}}

collection-stats:
    go run ./cmd/rb collection-stats -catalog-file cards/cards.json -collection-file collection/collection.json

missing *filters:
    # filters are set labels and domains, in any order: `just missing VEN chaos`
    go run ./cmd/rb missing -catalog-file cards/cards.json -collection-file collection/collection.json {{filters}}

# the rulings block stops to review any rulings not yet decided on: `just gen-anki -only rulings`
gen-anki *flags:
    go run ./cmd/rb gen-anki -catalog-file cards/cards.json -image-dir cards/images -rulings-file rules/rulings.json -review-file rules/anki-review.json -out anki/ {{flags}}

# copy the generated card images into Anki's media folder, before importing the decks
add-anki-media *flags:
    go run ./cmd/rb add-anki-media -media-dir anki/media {{flags}}

# save every decklist copied to the clipboard, until ctrl+c
watch-decks *flags:
    go run ./cmd/rb watch-decks -catalog-file cards/cards.json -sets-file cards/sets.json -decks-file decks/decks.json {{flags}}

# find the cards each ruling's question names, for the rules tab to show
gen-rules-cards:
    go run ./cmd/rb gen-rules-cards -rulings-file rules/rulings.json -catalog-file cards/cards.json -out rules/ruling-cards.json

validate set="":
    go run ./cmd/rb validate -collection-file collection/collection.json {{set}}

