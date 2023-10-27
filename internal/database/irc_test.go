package database

import (
	"context"
	"fmt"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func getMockIrcChannel() domain.IrcChannel {
	return domain.IrcChannel{
		ID:         0,
		Enabled:    true,
		Name:       "ab_announcement",
		Password:   "password123",
		Detached:   true,
		Monitoring: false,
	}
}

func getMockIrcNetwork() domain.IrcNetwork {
	connectedSince := time.Now().Add(-time.Hour) // Example time 1 hour ago
	return domain.IrcNetwork{
		ID:      0,
		Name:    "Freenode",
		Enabled: true,
		Server:  "irc.freenode.net",
		Port:    6667,
		TLS:     true,
		Pass:    "serverpass",
		Nick:    "nickname",
		Auth: domain.IRCAuth{
			Mechanism: domain.IRCAuthMechanismSASLPlain,
			Account:   "useraccount",
			Password:  "userpassword",
		},
		InviteCommand: "INVITE",
		UseBouncer:    true,
		BouncerAddr:   "bouncer.freenode.net",
		Channels: []domain.IrcChannel{
			getMockIrcChannel(),
		},
		Connected:      true,
		ConnectedSince: &connectedSince,
	}
}

func TestIrcRepo_StoreNetwork(t *testing.T) {
	for _, dbType := range getDbs() {
		db := SetupDatabase(t, dbType)
		defer func(db *DB) {
			err := db.Close()
			if err != nil {
				t.Fatalf("Could not close db connection: %v", err)
			}
		}(db)
		log := SetupLogger()

		repo := NewIrcRepo(log, db)

		mockData := getMockIrcNetwork()

		t.Run(fmt.Sprintf("StoreNetwork_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			assert.NotNil(t, mockData)

			// Execute
			err := repo.StoreNetwork(context.Background(), &mockData)
			assert.NoError(t, err)

			// Verify
			assert.NotEqual(t, int64(0), mockData.ID)

			// Cleanup
			_ = repo.DeleteNetwork(context.Background(), int64(int(mockData.ID)))
		})
	}
}

func TestIrcRepo_StoreChannel(t *testing.T) {
	for _, dbType := range getDbs() {
		db := SetupDatabase(t, dbType)
		defer func(db *DB) {
			err := db.Close()
			if err != nil {
				t.Fatalf("Could not close db connection: %v", err)
			}
		}(db)
		log := SetupLogger()

		repo := NewIrcRepo(log, db)

		mockNetwork := getMockIrcNetwork()
		mockChannel := getMockIrcChannel()

		t.Run(fmt.Sprintf("StoreChannel_Insert_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreNetwork(context.Background(), &mockNetwork)
			assert.NoError(t, err)

			// Execute
			err = repo.StoreChannel(context.Background(), mockNetwork.ID, &mockChannel)
			assert.NoError(t, err)

			// Verify
			assert.NotEqual(t, int64(0), mockChannel.ID)

			// No need to clean up, since the test below will delete the network
		})

		t.Run(fmt.Sprintf("StoreChannel_Update_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreChannel(context.Background(), mockNetwork.ID, &mockChannel)
			assert.NoError(t, err)

			// Update mockChannel fields
			mockChannel.Enabled = false
			mockChannel.Name = "updated_name"

			// Execute
			err = repo.StoreChannel(context.Background(), mockNetwork.ID, &mockChannel)
			assert.NoError(t, err)

			// Verify
			fetchedChannel, fetchErr := repo.ListChannels(mockNetwork.ID)
			assert.NoError(t, fetchErr)
			assert.Equal(t, mockChannel.Enabled, fetchedChannel[0].Enabled)
			assert.Equal(t, mockChannel.Name, fetchedChannel[0].Name)

			// Cleanup
			_ = repo.DeleteNetwork(context.Background(), mockNetwork.ID)
		})
	}
}
