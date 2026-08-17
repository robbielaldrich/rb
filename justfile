out_dir := "riftcodex_cards"

# Download all card data from the Riftcodex API into riftcodex_cards/ (gitignored).
riftcodex-dl:
    go run ./cmd/riftcodex-dl -out {{out_dir}}
