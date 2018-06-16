package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"sync"

	_ "github.com/lib/pq"
	uuid "github.com/satori/go.uuid"

	"github.com/apex/log"
	"github.com/ngerakines/teamprops"
	"github.com/nlopes/slack"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use: "teamprops",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use the help command to list all commands.")
	},
}

var serverCmd = &cobra.Command{
	Use: "server",
	RunE: func(cmd *cobra.Command, args []string) error {

		logger := logrus.New()
		logger.Level = logrus.DebugLevel

		db, err := sql.Open("postgres", viper.GetString("DB"))
		if err != nil {
			return err
		}
		defer db.Close()

		stop := make(chan struct{})
		var wg sync.WaitGroup

		token := viper.GetString("token")
		if token == "" {
			return fmt.Errorf("token not provided")
		}

		connectionKey, err := uuid.NewV4()
		if token == "" {
			return err
		}

		slackListener := &teamprops.SlackServer{
			DB:            db,
			API:           slack.New(token),
			Logger:        logger,
			Stop:          make(chan struct{}),
			ConnectionKey: connectionKey.String(),
		}

		wg.Add(1)
		go func() {
			if err := teamprops.RunManagedRoutine(slackListener, stop, &wg); err != nil {
				log.WithError(err).Error("Error shutting down slack listenr.")
			} else {
				log.Info("slack listenr stopped.")
			}
			wg.Done()
		}()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		<-c
		close(stop)
		wg.Wait()
		log.Info("teamprops stopped")
		return nil
	},
}

func main() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().String("token", "", "Set the slack token to use.")

	viper.AutomaticEnv()
	viper.BindPFlag("token", serverCmd.Flags().Lookup("token"))

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
