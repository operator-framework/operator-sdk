package generator

// toPlural makes "input" word plural.
// TODO: make this more grammatically correct for special nouns.
func toPlural(input string) string {
	lastchar := input[len(input)-1:]

	if lastchar == "s" {
		return input + "es"
	}

	return input + "s"
}
