package teamprops

import (
	"fmt"
	"math/rand"
	"strings"
)

type themeInfo struct {
	id        int
	message   string
	reactions []string
}

var themes []themeInfo
var themeCount int
var availableReactions []string

func init() {
	availableReactions = []string{
		"thumbsup",
		"fire",
		"hot_pepper",
		"handshake",
		"ok_hand",
		"the_horns",
		"muscle",
		"heart",
		"sunglasses",
		"smile",
		"tada",
		"clap",
		"partyparrot",
		"joy",
		"star-struck",
		"bomb",
		"boom",
		"dizzy",
		"love_letter",
		"gem",
		"confetti_ball",
		"gift",
		"medal",
		"trophy",
		"sports_medal",
		"first_place_medal",
		"rocket",
		"rainbow",
		"white_check_mark",
		"bangbang",
		"100",
	}
	themes = []themeInfo{
		themeInfo{
			id:        1,
			message:   "You're the bomb dot com, {target}!",
			reactions: []string{"fire", "bomb"},
		},
		themeInfo{
			id:        2,
			message:   "Hot stuff, {target}!",
			reactions: []string{"fire", "hot_pepper"},
		},
		themeInfo{
			id:        3,
			message:   "Keep it going, {target}!",
			reactions: []string{"fire"},
		},
		themeInfo{
			id:        4,
			message:   "Great job, {target}!",
			reactions: []string{"fire"},
		},
		themeInfo{
			id:        5,
			message:   "I bet you sweat glitter, {target}. Great work!",
			reactions: []string{},
		},
		themeInfo{
			id:        6,
			message:   "You're more fun than bubble wrap, {target}!",
			reactions: []string{},
		},
		themeInfo{
			id:        7,
			message:   "I like your style, {target}!",
			reactions: []string{},
		},
		themeInfo{
			id:        8,
			message:   "You've inspired us all, {target}!",
			reactions: []string{},
		},
		themeInfo{
			id:        9,
			message:   "You're a smark cookie, {target}.",
			reactions: []string{},
		},
		themeInfo{
			id:        10,
			message:   "When you say, \"I meant to do that,\" we totally believe you, {target}.",
			reactions: []string{},
		},
		themeInfo{
			id:        11,
			message:   "You sure are great at figuring stuff out, {target}.",
			reactions: []string{},
		},
		themeInfo{
			id:        12,
			message:   "I bet you do crossword puzzles in ink, {target}.",
			reactions: []string{},
		},
		themeInfo{
			id:        13,
			message:   "You are making a difference, {target}!",
			reactions: []string{},
		},
		themeInfo{
			id:        14,
			message:   "Actions speak louder than words, and yours tell an incredible story, {target}.",
			reactions: []string{},
		},
		themeInfo{
			id:        15,
			message:   "Being around you makes everything better, {target}.",
			reactions: []string{},
		},
		themeInfo{
			id:        16,
			message:   "You're a gift to those around you, {target}.",
			reactions: []string{},
		},
		themeInfo{
			id:        17,
			message:   "You're doing great, {target}.",
			reactions: []string{},
		},
		themeInfo{
			id:        18,
			message:   "You're all that and a super-size bag of chips, {target}.",
			reactions: []string{},
		},
		themeInfo{
			id:        19,
			message:   "On a scale from 1 to 10, you're an 11, {target}.",
			reactions: []string{},
		},
		themeInfo{
			id:        20,
			message:   "You're better than a triple-scoop ice cream cone, {target}. With sprinkles.",
			reactions: []string{},
		},
		themeInfo{
			id:        21,
			message:   "There's ordinary, and then there's you, {target}.",
			reactions: []string{},
		},
		themeInfo{
			id:        22,
			message:   "You're even better than a unicorn because you're real.",
			reactions: []string{},
		},
	}
	themeCount = len(themes)
}

func selectTheme() themeInfo {
	return themes[rand.Intn(themeCount)]
}

func (i themeInfo) FullMessage(users []string) string {
	last := len(users)
	if last == 0 {
		return strings.Replace(i.message, "{target}", "everyone", 1)
	}
	if last == 1 {
		return strings.Replace(i.message, "{target}", users[0], 1)
	}
	if last == 2 {
		return strings.Replace(i.message, "{target}", fmt.Sprintf("%s and %s", users[0], users[1]), 1)
	}
	result := ""
	for i, user := range users {
		if i > 0 {
			if i == last-1 {
				result += ", and " + user
			} else {
				result += ", " + user
			}
		} else {
			result += user
		}
	}
	return strings.Replace(i.message, "{target}", result, 1)
}

func getReactions() []string {
	i := rand.Perm(len(availableReactions))
	return []string{
		availableReactions[i[0]],
		availableReactions[i[1]],
		availableReactions[i[2]],
	}
}
