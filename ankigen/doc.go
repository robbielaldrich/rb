// Package ankigen turns the card catalog into a deck for Anki
// (https://apps.ankiweb.net). It writes Anki's plain-text import file
// alongside the media it references, rather than a packaged .apkg, so that
// generating a deck needs no SQLite database of its own.
package ankigen
