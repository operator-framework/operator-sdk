package generator

// toPlural makes "input" word plural.
// TODO: make this an input parameter as English grammar is highly variable
func toPlural(input string) string {
	lastchar := input[len(input)-1:]

	if lastchar == "s" {
		return input + "es"
	} else if lastchar == "x" {
		return input + "es"
	} else if lastchar == "y" {
		return input[0:len(input)-2] + "ies"
	}

	lasttwo := input[len(input)-2:]

	if lasttwo == "ch" {
		return input + "es"
	} else if lasttwo == "sh" {
		return input + "es"
	}

	return input + "s"
}
