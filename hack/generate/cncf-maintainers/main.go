package main

import (
	"log"
	"os"

	"k8s.io/utils/set"
	"sigs.k8s.io/yaml"
)

type ownersMap map[string][]string
type aliasesMap struct {
	Aliases map[string][]string `json:"aliases,omitempty"`
}

func main() {
	ownersData, err := os.ReadFile("OWNERS")
	if err != nil {
		log.Fatal(err)
	}
	aliasData, err := os.ReadFile("OWNERS_ALIASES")
	if err != nil {
		log.Fatal(err)
	}

	var owners ownersMap
	if err := yaml.Unmarshal(ownersData, &owners); err != nil {
		log.Fatal(err)
	}

	var aliases aliasesMap
	if err := yaml.Unmarshal(aliasData, &aliases); err != nil {
		log.Fatal(err)
	}

	expandedOwners := make(map[string]set.Set[string])
	for group, ownersAliases := range owners {
		expandedOwners[group] = set.New[string]()
		for _, alias := range ownersAliases {
			if members, ok := aliases.Aliases[alias]; ok {
				expandedOwners[group].Insert(members...)
			} else {
				log.Fatalf("alias %q is listed in OWNERS group %q but was not found in OWNERS_ALIASES", alias, group)
			}
		}
	}

	outOwners := make(map[string][]string)
	for g, m := range expandedOwners {
		outOwners[g] = m.SortedList()
	}

	out, err := yaml.Marshal(outOwners)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(".cncf-maintainers", out, 0644); err != nil {
		log.Fatal(err)
	}
}
