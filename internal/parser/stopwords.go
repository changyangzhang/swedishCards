package parser

// SwedishStopwords are tokens to skip when picking a cloze target.
// Lowercased.
var SwedishStopwords = map[string]bool{
	"jag": true, "du": true, "han": true, "hon": true, "vi": true, "ni": true, "de": true,
	"det": true, "den": true, "dem": true,
	"är": true, "har": true, "hade": true, "var": true, "kan": true, "ska": true, "skall": true, "vill": true, "gör": true, "gjorde": true,
	"en": true, "ett": true, "och": true, "eller": true, "men": true, "som": true, "att": true, "med": true,
	"på": true, "i": true, "av": true, "till": true, "för": true, "om": true, "så": true, "inte": true,
	"min": true, "din": true, "sin": true, "mitt": true, "ditt": true, "sitt": true,
	"mina": true, "dina": true, "sina": true,
	"vad": true, "när": true, "vem": true, "hur": true,
	"this": true, "that": true, // belt-and-braces if English leaks in
}

// pronouns we use to detect sentence-ness.
var swedishPronouns = map[string]bool{
	"jag": true, "du": true, "han": true, "hon": true, "vi": true, "ni": true, "de": true,
	"det": true, "den": true,
}

// auxiliary/copula verbs that mark sentence-ness even without pronoun
var swedishVerbs = map[string]bool{
	"är": true, "har": true, "var": true, "kan": true, "ska": true, "vill": true, "gör": true,
}
