package teamprops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUsersFromMessage(t *testing.T) {
	var a = usersFromMessage("")
	assert.Len(t, a, 0, "The input contained no users.")

	var b = usersFromMessage("props <@U5H48257Y> for being great.")
	assert.Len(t, b, 1, "The input contained one user.")
	assert.Equal(t, "<@U5H48257Y>", b[0])

	var c = usersFromMessage("props test17 <@U5H48257Y> and <@UB4CD44TT>")
	assert.Len(t, c, 2, "The input contained two users.")
	assert.Equal(t, "<@U5H48257Y>", c[0])
	assert.Equal(t, "<@UB4CD44TT>", c[1])
}

func TestStuff(t *testing.T) {
	theme := themeInfo{
		message: "cool {target}",
	}

	assert.Equal(t, "cool nick", theme.FullMessage([]string{"nick"}))
	assert.Equal(t, "cool nick and ashley", theme.FullMessage([]string{"nick", "ashley"}))
	assert.Equal(t, "cool nick, ashley, and tim", theme.FullMessage([]string{"nick", "ashley", "tim"}))
}
