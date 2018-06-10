# About

teamprops is a slack bot that records props given to team members. From this, reports can be generated.

# Build

This project uses dep.

    $ dep ensure

This project uses [mage](https://github.com/magefile/mage).

    $ mage

Alternatively, you can build the cmd/teamprops/main.go source file by hand.

# Configuration

* TOKEN -- A Slack token for the application to use when interacting with your workspace.
* DB -- A Postgres SQL connection string used to connect to the database.
