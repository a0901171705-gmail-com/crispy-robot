package parser

import "strings"

type Script struct {
	Interactions []Interaction
}

type Interaction interface {
	sealedInteraction()
}

type Invocation struct {
	Cmd  string
	Args []string
}

func (i Invocation) sealedInteraction() {}

type Expectation struct {
	Content string
}

func (i Expectation) sealedInteraction() {}

type buildingExpectation struct {
	building bool
	contents strings.Builder
}

type Action interface {
	Interaction
	sealedAction()
}

type Select struct {
	Option string
}

func (s Select) sealedInteraction() {}
func (s Select) sealedAction()      {}

type Say struct {
	Text string
}

func (s Say) sealedInteraction() {}
func (s Say) sealedAction()      {}

var expectationDelimeter = "---"

func Parse(s string) (Script, error) {
	var interactions []Interaction
	// TODO: Handle windows line endings
	currentExpectation := buildingExpectation{}
	for _, line := range strings.Split(s, "\n") {
		if len(line) == 0 {
			continue
		}

		if line == expectationDelimeter {
			// Start the expectation block
			if !currentExpectation.building {
				currentExpectation.building = true
				continue
			}
			// Finish the expectation block
			interactions = append(interactions, Expectation{
				Content: currentExpectation.contents.String(),
			})
			currentExpectation = buildingExpectation{}
			continue
		}

		tokens := strings.Split(line, " ")
		// Write next line of expectation block
		if currentExpectation.building {
			if tokens[len(tokens)-1] == "|" {
				currentExpectation.contents.WriteString(strings.TrimRight(line, "|"))
			} else {
				currentExpectation.contents.WriteString(line + "\n")
			}
			continue
		}

		if tokens[0] == "select" {
			interactions = append(interactions, Select{
				Option: strings.Join(tokens[1:], " "),
			})
			continue
		}

		if tokens[0] == "say" {
			interactions = append(interactions, Say{
				Text: strings.Join(tokens[1:], " "),
			})
			continue
		}

		interactions = append(
			interactions,
			Invocation{
				Cmd:  tokens[0],
				Args: tokens[1:],
			},
		)
	}

	return Script{Interactions: interactions}, nil
}
