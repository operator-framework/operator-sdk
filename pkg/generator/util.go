package generator

// toPlural makes "input" word plural.
// TODO: make this more grammatically correct for special nouns.
func toPlural(input string) string {
	return input + "s"
}
