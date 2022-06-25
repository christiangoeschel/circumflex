package comment

import (
	"clx/comment/postprocessor"
	"clx/constants/margins"
	"clx/constants/unicode"
	"clx/item"
	"clx/meta"
	"clx/parser"
	"clx/settings"
	"clx/syntax"
	"strconv"
	"strings"

	"github.com/logrusorgru/aurora/v3"

	text "github.com/MichaelMure/go-term-text"
)

const (
	newLine      = "\n"
	newParagraph = "\n\n"
)

func ToString(comments *item.Item, config *settings.Config, screenWidth int, lastVisited int64) string {
	commentSectionScreenWidth := screenWidth - margins.CommentSectionLeftMargin

	header := getHeader(comments, config, lastVisited)
	firstCommentID := getFirstCommentID(comments.Comments)

	replies := ""

	for _, reply := range comments.Comments {
		replies += printReplies(reply, config, commentSectionScreenWidth, comments.User, "", firstCommentID,
			lastVisited)
	}

	commentSection := postprocessor.Process(header+replies+newLine, screenWidth)

	return commentSection
}

func getFirstCommentID(comments []*item.Item) int {
	if len(comments) == 0 {
		return 0
	}

	return comments[0].ID
}

func getHeader(c *item.Item, config *settings.Config, lastVisited int64) string {
	newComments := getNewCommentsCount(c, lastVisited)

	return meta.GetCommentSectionMetaBlock(c, config, newComments) + newParagraph
}

func printReplies(c *item.Item, config *settings.Config, screenWidth int, originalPoster string,
	parentPoster string, firstCommentID int, lastVisited int64) string {
	isDeletedAndHasNoReplies := c.Content == "[deleted]" && len(c.Comments) == 0
	if isDeletedAndHasNoReplies {
		return ""
	}

	indentation := getIndentString(c.Level)
	indentSize := len(indentation)
	availableScreenWidth := screenWidth - indentSize - margins.CommentSectionLeftMargin
	adjustedCommentWidth := config.CommentWidth - c.Level

	comment := formatComment(c, config, originalPoster, parentPoster, adjustedCommentWidth, availableScreenWidth,
		lastVisited)
	indentedComment, _ := text.WrapWithPad(comment, screenWidth, indentation)
	fullComment := getSeparator(c.Level, config.CommentWidth, c.ID, firstCommentID) + indentedComment + newLine

	if c.Level == 0 {
		parentPoster = c.User
	}

	for _, reply := range c.Comments {
		fullComment += printReplies(reply, config, screenWidth, originalPoster, parentPoster, firstCommentID,
			lastVisited)
	}

	return fullComment
}

func formatComment(c *item.Item, config *settings.Config, originalPoster string, parentPoster string, commentWidth int,
	availableScreenWidth int, lastVisited int64) string {
	coloredIndentSymbol := syntax.ColorizeIndentSymbol(config.IndentationSymbol, c.Level)

	header := getCommentHeader(c, originalPoster, parentPoster, commentWidth, lastVisited)
	comment := parser.ParseComment(c.Content, config, commentWidth, availableScreenWidth)

	paddedComment, _ := text.WrapWithPad(comment, availableScreenWidth, coloredIndentSymbol)

	return header + paddedComment
}

func getSeparator(level int, commentWidth int, currentCommentID int, firstCommentID int) string {
	if currentCommentID == firstCommentID {
		return ""
	}

	if level != 0 || currentCommentID == firstCommentID {
		return newLine
	}

	return aurora.Faint(strings.Repeat(" ", commentWidth)).Underline().String() + newLine + newLine
}

func getIndentString(level int) string {
	if level == 0 {
		return ""
	}

	return strings.Repeat(" ", level-1)
}

func getCommentHeader(c *item.Item, originalPoster string, parentPoster string, commentWidth int, lastVisited int64) string {
	if c.Level == 0 {
		return formatHeader(c, originalPoster, parentPoster, true, true, true,
			commentWidth, 0, lastVisited)
	}

	return formatHeader(c, originalPoster, parentPoster, false, false, false,
		commentWidth, 1, lastVisited)
}

func formatHeader(c *item.Item, originalPoster string, parentPoster string, underlineHeader bool, showReplies bool,
	enableZeroWidthSpace bool, commentWidth int, indentSize int, lastVisited int64) string {
	author := getAuthor(c.User, lastVisited, c.Time)
	authorLabel := getAuthorLabel(c.User, originalPoster, parentPoster)
	zeroWidthSpace := getZeroWidthSpace(enableZeroWidthSpace)
	repliesTag := getReplies(showReplies, c)
	indentation := strings.Repeat(" ", indentSize)

	spacingLength := commentWidth - text.Len(indentation+author+authorLabel+c.TimeAgo+repliesTag)
	spacing := strings.Repeat(" ", spacingLength)

	return zeroWidthSpace + indentation + author + authorLabel +
		underlineAndDim(underlineHeader, c.TimeAgo, spacing, repliesTag) + newLine
}

func getAuthor(author string, lastVisited, timePosted int64) string {
	authorInBold := aurora.Bold(author).String() + " "

	commentIsNew := lastVisited < timePosted

	if commentIsNew {
		return authorInBold + aurora.Blue("●").String() + " "
	}

	return authorInBold
}

func underlineAndDim(enabled bool, timeAgo, spacing, replies string) string {
	if enabled {
		return aurora.Faint(timeAgo + spacing + replies).String()
	}

	return aurora.Faint(timeAgo).String()
}

func getReplies(showReplies bool, children *item.Item) string {
	if !showReplies {
		return ""
	}

	numberOfReplies := getReplyCount(children)
	replyTag := getRepliesTag(numberOfReplies)

	return replyTag
}

func getRepliesTag(numberOfReplies int) string {
	if numberOfReplies == 0 {
		return ""
	}

	return strconv.Itoa(numberOfReplies) + " ↩"
}

func getZeroWidthSpace(enabled bool) string {
	if enabled {
		return unicode.ZeroWidthSpace
	}

	return ""
}

func getReplyCount(comments *item.Item) int {
	numberOfReplies := 0

	return incrementReplyCount(comments, &numberOfReplies)
}

func incrementReplyCount(comments *item.Item, repliesSoFar *int) int {
	for _, reply := range comments.Comments {
		*repliesSoFar++
		incrementReplyCount(reply, repliesSoFar)
	}

	return *repliesSoFar
}

func getNewCommentsCount(comments *item.Item, lastVisited int64) int {
	numberOfReplies := 0

	return incrementNewCommentsCount(comments, &numberOfReplies, lastVisited)
}

func incrementNewCommentsCount(comments *item.Item, newCommentsSoFar *int, lastVisited int64) int {
	for _, reply := range comments.Comments {
		commentIsNew := lastVisited < reply.Time
		if commentIsNew {
			*newCommentsSoFar++
		}

		incrementNewCommentsCount(reply, newCommentsSoFar, lastVisited)
	}

	return *newCommentsSoFar
}

func getAuthorLabel(author, originalPoster, parentPoster string) string {
	switch author {
	case "dang":
		return aurora.Green("mod ").String()

	case originalPoster:
		return aurora.Red("OP ").String()

	case parentPoster:
		return aurora.Magenta("PP ").String()

	default:
		return ""
	}
}