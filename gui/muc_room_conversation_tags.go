package gui

import (
	"github.com/coyim/gotk3adapter/gtki"
	"github.com/coyim/gotk3adapter/pangoi"
)

func (c *roomViewConversation) initTagsAndTextBuffer(v *roomView) {
	c.tags = c.newMUCTableStyleTags(v.u)

	buf, _ := g.gtk.TextBufferNew(c.tags)
	c.text.SetBuffer(buf)
}

func (c *roomViewConversation) createConversationTag(name string, properties map[string]interface{}) gtki.TextTag {
	tag, _ := g.gtk.TextTagNew(name)
	for attribute, value := range properties {
		_ = tag.SetProperty(attribute, value)
	}
	return tag
}

func (c *roomViewConversation) createWarningTag(cs mucColorSet) gtki.TextTag {
	return c.createConversationTag("warning", map[string]interface{}{
		"foreground": cs.warningForeground,
	})
}

func (c *roomViewConversation) createLeftRoomTag(cs mucColorSet) gtki.TextTag {
	return c.createConversationTag("leftRoom", map[string]interface{}{
		"foreground": cs.someoneLeftForeground,
		"style":      pangoi.STYLE_ITALIC,
	})
}

func (c *roomViewConversation) createJoinedRoomTag(cs mucColorSet) gtki.TextTag {
	return c.createConversationTag("joinedRoom", map[string]interface{}{
		"foreground": cs.someoneJoinedForeground,
		"style":      pangoi.STYLE_ITALIC,
	})
}

func (c *roomViewConversation) createTimestampTag(cs mucColorSet) gtki.TextTag {
	return c.createConversationTag("timestamp", map[string]interface{}{
		"foreground": cs.timestampForeground,
		"style":      pangoi.STYLE_NORMAL,
	})
}

func (c *roomViewConversation) createNicknameTag(cs mucColorSet) gtki.TextTag {
	return c.createConversationTag("nickname", map[string]interface{}{
		"foreground": cs.nicknameForeground,
		"style":      pangoi.STYLE_NORMAL,
	})
}

func (c *roomViewConversation) createSubjectTag(cs mucColorSet) gtki.TextTag {
	return c.createConversationTag("subject", map[string]interface{}{
		"foreground": cs.subjectForeground,
		"style":      pangoi.STYLE_ITALIC,
	})
}

func (c *roomViewConversation) createMessageTag(cs mucColorSet) gtki.TextTag {
	return c.createConversationTag("message", map[string]interface{}{
		"foreground": cs.messageForeground,
		"style":      pangoi.STYLE_NORMAL,
	})
}

func (c *roomViewConversation) newMUCTableStyleTags(u *gtkUI) gtki.TextTagTable {
	table, _ := g.gtk.TextTagTableNew()
	cs := u.currentMUCColorSet()

	tags := []func(mucColorSet) gtki.TextTag{
		c.createWarningTag,
		c.createLeftRoomTag,
		c.createJoinedRoomTag,
		c.createTimestampTag,
		c.createNicknameTag,
		c.createSubjectTag,
		c.createMessageTag,
	}

	for _, t := range tags {
		table.Add(t(cs))
	}

	return table
}