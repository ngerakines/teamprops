package teamprops

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/kr/pretty"
	"github.com/nlopes/slack"
	"github.com/sirupsen/logrus"
)

type SlackServer struct {
	API    *slack.Client
	DB     *sql.DB
	Logger *logrus.Logger

	Stop chan struct{}

	myID string
}

func (s *SlackServer) Run() error {
	rtm := s.API.NewRTM()
	go rtm.ManageConnection()

	for {
		select {
		case <-s.Stop:
			return nil
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.LatencyReport:
				s.Logger.WithField("time", ev.Value).Debug("Latency report")
				break

			case *slack.HelloEvent:
				s.Logger.Debug("Hello")
				break

			case *slack.ConnectingEvent:
				s.Logger.
					WithFields(logrus.Fields{
						"counter": ev.ConnectionCount,
						"attempt": ev.Attempt,
					}).
					Debug("Connecting")
				break

			case *slack.ConnectedEvent:
				s.Logger.WithField("counter", ev.ConnectionCount).Info("Connected")

				info := rtm.GetInfo()
				s.myID = info.User.ID

				for _, channel := range ev.Info.Channels {
					if channel.IsChannel && channel.IsMember {
						s.Logger.WithField("channel", channel.Name).Info("Listening to channel.")
					}
				}
				break

			case *slack.ChannelCreatedEvent:
				s.Logger.WithField("channel", ev.Channel.Name).Info("Channel created")
				break

			case *slack.UserTypingEvent:
				s.Logger.
					WithFields(logrus.Fields{
						"channel": ev.Channel,
						"user":    ev.User,
					}).
					Debug("User typing")
				break

			case *slack.MessageEvent:
				s.Logger.
					WithFields(logrus.Fields{
						"channel": ev.Channel,
						"user":    ev.User,
						"message": ev.Msg.Text,
					}).
					Debug("Message received")

				if ev.User == s.myID {
					s.Logger.Info("Ignoring one of my own messages.")
				}

				if isProps(ev.Msg.Text) {
					theme := selectTheme()
					users := usersFromMessage(ev.Msg.Text)
					messageID, err := s.createProps(ev.Msg.Channel, ev.Msg.User, ev.Msg.Timestamp, ev.Msg.Text, theme.id)
					if err != nil {
						s.Logger.WithError(err).Error("Error creating props record.")
					}
					rtm.SendMessage(&slack.OutgoingMessage{
						ID:      messageID,
						Channel: ev.Channel,
						Text:    theme.FullMessage(users),
						Type:    "message",
					})
				}
				break

			case *slack.AckMessage:
				if err := s.updatePropsReply(ev.ReplyTo, ev.Timestamp); err != nil {
					s.Logger.WithError(err).Error("Error recording reply.")
					break
				}

				var channel string
				var theme int
				err := s.DB.QueryRow("SELECT target_channel, theme FROM props WHERE id = $1", ev.ReplyTo).Scan(&channel, &theme)
				if err != nil {
					s.Logger.WithError(err).Error("Error recording reply.")
					break
				}

				messageTheme := themes[theme]
				if len(messageTheme.reactions) > 0 {
					for _, reaction := range messageTheme.reactions {
						rtm.AddReaction(reaction, slack.ItemRef{
							Channel:   channel,
							Timestamp: ev.Timestamp,
						})
					}
				} else {
					for _, reaction := range getReactions() {
						rtm.AddReaction(reaction, slack.ItemRef{
							Channel:   channel,
							Timestamp: ev.Timestamp,
						})
					}
				}
				break

			case *slack.ReactionAddedEvent:
				if ev.ItemUser != s.myID {
					s.Logger.Debug("Item added to someone else's event.")
					break
				}
				if ev.User == s.myID {
					s.Logger.Debug("This is my own automated reaction.")
					break
				}

				if err := s.createReaction(ev.Item.Channel, ev.User, ev.Item.Timestamp, ev.Reaction); err != nil {
					s.Logger.WithError(err).Error("Error recording reaction.")
				}

				break

			case *slack.ReactionRemovedEvent:
				if ev.ItemUser != s.myID {
					s.Logger.Debug("Item removed from someone else's event.")
					break
				}

				if err := s.removeReaction(ev.Item.Channel, ev.User, ev.Item.Timestamp, ev.Reaction); err != nil {
					s.Logger.WithError(err).Error("Error removing reaction.")
				}

				break

			case *slack.RTMError:
				s.Logger.WithError(ev).Error()
				break

			case *slack.InvalidAuthEvent:
				s.Logger.Error("Invalid credentials")
				return fmt.Errorf("invalid credentials")

			case *slack.ChannelJoinedEvent:
				s.Logger.WithField("channel", ev.Channel.Name).Info("Joined channel")
				break

			case *slack.ChannelLeftEvent:
				s.Logger.WithField("channel", ev.Channel).Info("Left channel")
				break

			case *slack.PrefChangeEvent:
				break

			default:
				pretty.Println(msg)
			}
		}
	}
}

func (s *SlackServer) Shutdown(ctx context.Context) error {
	close(s.Stop)
	return nil
}

func (s *SlackServer) createProps(channel, author, timestamp, message string, theme int) (int, error) {
	query := `INSERT INTO
		props
			(source_author, source_timestamp, source_message, source_channel, target_channel, theme)
		VALUES
			($1, $2, $3, $4, $5, $6)
		RETURNING id`

	stmt, err := s.DB.Prepare(query)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var messageID int
	err = stmt.QueryRow(author, timestamp, message, channel, channel, theme).Scan(&messageID)
	if err != nil {
		return 0, err
	}
	return messageID, nil
}

func (s *SlackServer) updatePropsReply(id int, timestamp string) error {
	_, err := s.DB.Exec("UPDATE props SET target_timestamp = $1, updated_at = NOW() WHERE id = $2", timestamp, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *SlackServer) createReaction(channel, author, timestamp, reaction string) error {
	query := `INSERT INTO
		reactions
			(channel, message_timestamp, reaction_user, reaction)
		VALUES
			($1, $2, $3, $4)
		ON CONFLICT (channel, message_timestamp, reaction_user, reaction)
		DO UPDATE SET removed = false`

	_, err := s.DB.Exec(query, channel, timestamp, author, reaction)
	if err != nil {
		return err
	}
	return nil
}

func (s *SlackServer) removeReaction(channel, author, timestamp, reaction string) error {
	query := `UPDATE
		reactions
	SET
		removed = true
	WHERE
		channel = $1
		AND message_timestamp = $2
		AND reaction_user = $3
		AND reaction = $4`

	_, err := s.DB.Exec(query, channel, timestamp, author, reaction)
	if err != nil {
		return err
	}
	return nil
}

func isProps(input string) bool {
	prefixes := []string{"props", "kudos", "congrats"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(input, prefix) {
			return true
		}
	}
	return false
}

func usersFromMessage(input string) []string {
	re := regexp.MustCompile(`(<@[^>]+>)`)
	if matches := re.FindAllString(input, -1); matches != nil {
		return matches
	}

	return []string{}
}
