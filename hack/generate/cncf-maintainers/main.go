package main

import (
	"io/ioutil"
	"log"

	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/yaml"
)

type ownersMap map[string][]string
type aliasesMap struct {
	Aliases map[string][]string `json:"aliases,omitempty"`
}

func main() {
	ownersData, err := ioutil.ReadFile("OWNERS")
	if err != nil {
		log.Fatal(err)
	}
	aliasData, err := ioutil.ReadFile("OWNERS_ALIASES")
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

	expandedOwners := make(map[string]sets.String)
	for group, ownersAliases := range owners {
		expandedOwners[group] = sets.NewString()
		for _, alias := range ownersAliases {
			if members, ok := aliases.Aliases[alias]; ok {
				expandedOwners[group].Insert(members...)
			}
		}
	}

	outOwners := make(map[string][]string)
	for g, m := range expandedOwners {
		outOwners[g] = m.List()
	}

	out, err := yaml.Marshal(outOwners)
	if err != nil {
		log.Fatal(err)
	}

	if err := ioutil.WriteFile(".cncf-maintainers", out, 0644); err != nil {
		log.Fatal(err)
	}
}
