package ankigen

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"rb/rules"
)

// A drafted answer takes whole sentences until it has said something: a
// ruling that opens with a bare "No." needs the sentence after it, one that
// opens with a full sentence does not.
const minAnswer = 40

// reddit is the category of the hand-written thread notes, whose topic names
// the subreddit rather than a card or a rule.
const reddit = "reddit-field-notes"

// condense drafts a note from a ruling. Both halves come off the front of the
// ruling, which is written to lead with the answer and leave the reasoning to
// the paragraphs after it — a first cut for the review to reword, not a
// replacement for it.
func condense(r rules.Ruling) (front, back string) {
	return draftQuestion(r), draftAnswer(r)
}

// draftQuestion keeps the ruling's own question, which is already phrased as
// one, and names the topic in front of it when the question doesn't: "Can it
// be targeted?" is no use on a card standing on its own.
func draftQuestion(r rules.Ruling) string {
	q := plainText(r.Question)
	if q == "" || r.Category == reddit || mentions(q, r.Topic) {
		return q
	}
	return r.Topic + " — " + q
}

// mentions reports whether a question already names its topic. A card's topic
// is its full printed name while the question tends to use the short one, so
// "Akshan, Mischievous" counts as mentioned by a question asking about
// "Akshan".
func mentions(question, topic string) bool {
	name, _, _ := strings.Cut(strings.ToLower(topic), ",")
	return strings.Contains(strings.ToLower(question), strings.TrimSpace(name))
}

// draftAnswer takes the opening of the ruling, which states it; everything
// past the first blank line explains it instead.
func draftAnswer(r rules.Ruling) string {
	sentences := splitSentences(plainText(firstParagraph(r.Answer)))
	if len(sentences) == 0 {
		return ""
	}
	answer := sentences[0]
	for _, s := range sentences[1:] {
		if utf8.RuneCountInString(answer) >= minAnswer {
			break
		}
		answer += " " + s
	}
	return answer
}

func firstParagraph(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if i := strings.Index(s, "\n\n"); i >= 0 {
		return s[:i]
	}
	return s
}

var (
	mdLink  = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	spacing = regexp.MustCompile(`\s+`)
)

// plainText flattens the markdown the rulings are written in. An Anki field
// renders as HTML, so a link or a bold marker left in would show as its own
// source.
func plainText(s string) string {
	s = mdLink.ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, "**", "")
	return strings.TrimSpace(spacing.ReplaceAllString(s, " "))
}

// splitSentences breaks a paragraph after each sentence. A full stop ends one
// only when a capital follows it, which keeps rule numbers, decimals and
// abbreviations inside the sentence they belong to.
func splitSentences(p string) []string {
	var out []string
	start := 0
	for i := 0; i < len(p); i++ {
		if !strings.ContainsRune(".!?", rune(p[i])) {
			continue
		}
		end := i + 1
		for end < len(p) && strings.ContainsRune(`"')`, rune(p[end])) {
			end++
		}
		next := end
		for next < len(p) && p[next] == ' ' {
			next++
		}
		if next == end || next == len(p) || !opensSentence(p[next:]) {
			continue
		}
		out = append(out, strings.TrimSpace(p[start:end]))
		start, i = next, next-1
	}
	if tail := strings.TrimSpace(p[start:]); tail != "" {
		out = append(out, tail)
	}
	return out
}

// opensSentence reports whether s starts a sentence rather than continuing
// one. A ruling can open on a quoted phrase, so an opening quote is looked
// past rather than counted against it.
func opensSentence(s string) bool {
	for _, r := range s {
		if r == '"' || r == '\'' {
			continue
		}
		return unicode.IsUpper(r)
	}
	return false
}
