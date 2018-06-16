package teamprops

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/kr/pretty"
	"github.com/nlopes/slack"
	"github.com/sirupsen/logrus"
)

type SlackServer struct {
	API           *slack.Client
	DB            *sql.DB
	Logger        *logrus.Logger
	ConnectionKey string

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
					s.Logger.Debug("Ignoring one of my own messages.")
				}

				if isProps(ev.Msg.Text) {
					theme := selectTheme()
					users := usersFromMessage(ev.Msg.Text)
					out := rtm.NewOutgoingMessage(theme.FullMessage(users), ev.Channel)
					if err := s.createProps(s.ConnectionKey, ev.Msg.Channel, ev.Msg.User, ev.Msg.Timestamp, ev.Msg.Text, theme.id, out.ID); err != nil {
						s.Logger.WithError(err).Error("Error creating props record.")
						break
					}
					rtm.SendMessage(out)
					break
				}
				if s.isLeaderboard(ev.Msg.Text) {
					givePairs, err := s.topPropsGivers()
					if err != nil {
						s.Logger.WithError(err).Error("Error creating leaderboard")
						break
					}
					message := "Props given: \n"
					for i, pair := range givePairs {
						if i < 3 {
							message += fmt.Sprintf("%d. <@%s> %d\n", i+1, pair.Author, pair.Value)
						}
					}
					recvPairs, err := s.topPropsReceivers()
					if err != nil {
						s.Logger.WithError(err).Error("Error creating leaderboard")
						break
					}
					message += "Props received: \n"
					for i, pair := range recvPairs {
						if i < 3 {
							message += fmt.Sprintf("%d. <@%s> %d\n", i+1, pair.Author, pair.Value)
						}
					}
					rtm.SendMessage(rtm.NewOutgoingMessage(message, ev.Channel))
				}
				break

			case *slack.AckMessage:
				updated, err := s.updatePropsReply(s.ConnectionKey, ev.ReplyTo, ev.Timestamp)
				if err != nil {
					s.Logger.WithError(err).Error("Error recording reply.")
					break
				}
				if !updated {
					break
				}

				var channel string
				var theme int
				err = s.DB.QueryRow("SELECT target_channel, theme FROM props WHERE connection_key = $1 AND connection_id = $2", s.ConnectionKey, ev.ReplyTo).Scan(&channel, &theme)
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

func (s *SlackServer) createProps(connectionKey, channel, author, timestamp, message string, theme, messageID int) error {
	query := `INSERT INTO
		props
			(connection_key, connection_id, source_author, source_timestamp, source_message, source_channel, target_channel, theme)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8)`

	if _, err := s.DB.Exec(query, connectionKey, messageID, author, timestamp, message, channel, channel, theme); err != nil {
		return err
	}
	return nil
}

func (s *SlackServer) updatePropsReply(connectionKey string, messageID int, timestamp string) (bool, error) {
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM props WHERE connection_key = $1 AND connection_id = $2", connectionKey, messageID).Scan(&count)
	if err != nil {
		return false, err
	}
	if count == 0 {
		return false, nil
	}
	if _, err := s.DB.Exec("UPDATE props SET target_timestamp = $1, updated_at = NOW() WHERE connection_key = $2 AND connection_id = $3", timestamp, connectionKey, messageID); err != nil {
		return false, err
	}
	return true, nil
}

func (s *SlackServer) createReaction(channel, author, timestamp, reaction string) error {
	query := `INSERT INTO
		reactions
			(channel, message_timestamp, reaction_user, reaction)
		VALUES
			($1, $2, $3, $4)
		ON CONFLICT (channel, message_timestamp, reaction_user, reaction)
		DO UPDATE SET removed = false`

	if _, err := s.DB.Exec(query, channel, timestamp, author, reaction); err != nil {
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

	if _, err := s.DB.Exec(query, channel, timestamp, author, reaction); err != nil {
		return err
	}
	return nil
}

func isProps(input string) bool {
	prefixes := []string{"props", "kudos", "congrats"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(input), prefix) {
			return true
		}
	}
	return false
}

func (s *SlackServer) isLeaderboard(input string) bool {
	mention := fmt.Sprintf("<@%s>", s.myID)
	if strings.HasPrefix(input, mention) {
		return strings.Index(strings.ToLower(input), "leaderboard") > 0
	}
	return false
}

func (s *SlackServer) topPropsGivers() (AuthorMetricPairList, error) {
	query := `SELECT source_author, COUNT(*) AS count FROM props GROUP BY source_author;
	SELECT reaction_user, COUNT(*) AS count FROM reactions GROUP BY reaction_user;`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := map[string]int{}

	for rows.Next() {
		var (
			author string
			count  int
		)
		if err := rows.Scan(&author, &count); err != nil {
			return nil, err
		}
		results[author] = count
	}

	if !rows.NextResultSet() {
		return nil, fmt.Errorf("expected more database results")
	}

	for rows.Next() {
		var (
			author string
			count  int
		)
		if err := rows.Scan(&author, &count); err != nil {
			return nil, err
		}
		value, ok := results[author]
		if ok {
			results[author] = value + count
		} else {
			results[author] = count
		}
	}

	pairs := make(AuthorMetricPairList, 0, len(results))
	for author, count := range results {
		pairs = append(pairs, AuthorMetricPair{author, count})
	}
	sort.Sort(sort.Reverse(pairs))

	return pairs, nil
}

func (s *SlackServer) collectUsersFromMessages() (map[string][]string, error) {
	query := `SELECT target_timestamp, source_message FROM props`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := make(map[string][]string)

	for rows.Next() {
		var (
			timestamp string
			message   string
		)
		if err := rows.Scan(&timestamp, &message); err != nil {
			return nil, err
		}
		results[timestamp] = usersFromMessage(message)
	}
	return results, nil
}

func (s *SlackServer) topPropsReceivers() (AuthorMetricPairList, error) {
	messageUsers, err := s.collectUsersFromMessages()
	if err != nil {
		return nil, err
	}

	userProps := make(map[string]int)

	for _, users := range messageUsers {
		for _, user := range users {
			value, ok := userProps[user]
			if ok {
				userProps[user] = value + 1
			} else {
				userProps[user] = 1
			}
		}
	}

	query := `select message_timestamp, count(*) as count from reactions group by message_timestamp;`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			timestamp string
			count     int
		)
		if err := rows.Scan(&timestamp, &count); err != nil {
			return nil, err
		}
		if users, ok := messageUsers[timestamp]; ok {
			for _, user := range users {
				value, ok := userProps[user]
				if ok {
					userProps[user] = value + count
				} else {
					userProps[user] = 1
				}
			}
		}
	}

	pairs := make(AuthorMetricPairList, 0, len(userProps))
	for author, count := range userProps {
		pairs = append(pairs, AuthorMetricPair{author, count})
	}
	sort.Sort(sort.Reverse(pairs))

	return pairs, nil
}

func usersFromMessage(input string) []string {
	re := regexp.MustCompile(`(<@[^>]+>)`)
	if matches := re.FindAllString(input, -1); matches != nil {
		return matches
	}

	return []string{}
}
