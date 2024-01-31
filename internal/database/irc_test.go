// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
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
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

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
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

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

func TestIrcRepo_UpdateNetwork(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewIrcRepo(log, db)

		mockData := getMockIrcNetwork()

		t.Run(fmt.Sprintf("UpdateNetwork_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			assert.NotNil(t, mockData)
			err := repo.StoreNetwork(context.Background(), &mockData)
			assert.NoError(t, err)
			assert.NotEqual(t, int64(0), mockData.ID)

			// Update mockData fields
			mockData.Enabled = true
			mockData.Name = "UpdatedNetworkName"

			// Execute
			err = repo.UpdateNetwork(context.Background(), &mockData)
			assert.NoError(t, err)

			// Verify
			updatedNetwork, fetchErr := repo.GetNetworkByID(context.Background(), mockData.ID)
			assert.NoError(t, fetchErr)
			assert.Equal(t, mockData.Enabled, updatedNetwork.Enabled)
			assert.Equal(t, mockData.Name, updatedNetwork.Name)

			// Cleanup
			_ = repo.DeleteNetwork(context.Background(), mockData.ID)
		})
	}
}

func TestIrcRepo_GetNetworkByID(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewIrcRepo(log, db)

		mockData := getMockIrcNetwork()

		t.Run(fmt.Sprintf("GetNetworkByID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			assert.NotNil(t, mockData)
			err := repo.StoreNetwork(context.Background(), &mockData)
			assert.NoError(t, err)
			assert.NotEqual(t, int64(0), mockData.ID)

			// Execute
			fetchedNetwork, err := repo.GetNetworkByID(context.Background(), mockData.ID)
			assert.NoError(t, err)

			// Verify
			assert.NotNil(t, fetchedNetwork)
			assert.Equal(t, mockData.ID, fetchedNetwork.ID)
			assert.Equal(t, mockData.Enabled, fetchedNetwork.Enabled)
			assert.Equal(t, mockData.Name, fetchedNetwork.Name)

			// Cleanup
			_ = repo.DeleteNetwork(context.Background(), mockData.ID)
		})
	}
}

func TestIrcRepo_DeleteNetwork(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewIrcRepo(log, db)

		mockData := getMockIrcNetwork()

		t.Run(fmt.Sprintf("DeleteNetwork_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			assert.NotNil(t, mockData)
			err := repo.StoreNetwork(context.Background(), &mockData)
			assert.NoError(t, err)
			assert.NotEqual(t, int64(0), mockData.ID)

			// Execute
			err = repo.DeleteNetwork(context.Background(), mockData.ID)
			assert.NoError(t, err)

			// Verify
			fetchedNetwork, fetchErr := repo.GetNetworkByID(context.Background(), mockData.ID)
			assert.Error(t, fetchErr)
			assert.Nil(t, fetchedNetwork)
		})
	}
}

func TestIrcRepo_FindActiveNetworks(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewIrcRepo(log, db)

		mockData1 := getMockIrcNetwork()
		mockData1.Enabled = true

		mockData2 := getMockIrcNetwork()
		mockData2.Enabled = false
		// These fields are required to be unique
		mockData2.Server = "irc.example.com"
		mockData2.Port = 6664
		mockData2.Nick = "nickname2"

		t.Run(fmt.Sprintf("FindActiveNetworks_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreNetwork(context.Background(), &mockData1)
			assert.NoError(t, err)
			err = repo.StoreNetwork(context.Background(), &mockData2)
			assert.NoError(t, err)

			// Execute
			activeNetworks, err := repo.FindActiveNetworks(context.Background())
			assert.NoError(t, err)

			// Verify
			assert.NotEmpty(t, activeNetworks)
			assert.Len(t, activeNetworks, 1)
			assert.True(t, activeNetworks[0].Enabled)

			// Cleanup
			_ = repo.DeleteNetwork(context.Background(), mockData1.ID)
			_ = repo.DeleteNetwork(context.Background(), mockData2.ID)
		})
	}
}

func TestIrcRepo_ListNetworks(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewIrcRepo(log, db)

		// Prepare mock data
		mockData1 := getMockIrcNetwork()
		mockData1.Name = "ZNetwork"
		mockData2 := getMockIrcNetwork()
		mockData2.Name = "ANetwork"
		mockData2.Server = "irc.example.com"
		mockData2.Port = 6664
		mockData2.Nick = "nickname2"

		t.Run(fmt.Sprintf("ListNetworks_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreNetwork(context.Background(), &mockData1)
			assert.NoError(t, err)
			err = repo.StoreNetwork(context.Background(), &mockData2)
			assert.NoError(t, err)

			// Execute
			listedNetworks, err := repo.ListNetworks(context.Background())
			assert.NoError(t, err)

			// Verify
			assert.NotEmpty(t, listedNetworks)
			assert.Len(t, listedNetworks, 2)

			// Verify the order is alphabetical based on the name
			assert.Equal(t, "ANetwork", listedNetworks[0].Name)
			assert.Equal(t, "ZNetwork", listedNetworks[1].Name)

			// Cleanup
			_ = repo.DeleteNetwork(context.Background(), mockData1.ID)
			_ = repo.DeleteNetwork(context.Background(), mockData2.ID)
		})
	}
}

func TestIrcRepo_ListChannels(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewIrcRepo(log, db)
		mockNetwork := getMockIrcNetwork()
		mockChannel := getMockIrcChannel()

		t.Run(fmt.Sprintf("ListChannels_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreNetwork(context.Background(), &mockNetwork)
			assert.NoError(t, err)

			err = repo.StoreChannel(context.Background(), mockNetwork.ID, &mockChannel)
			assert.NoError(t, err)

			// Execute
			listedChannels, err := repo.ListChannels(mockNetwork.ID)
			assert.NoError(t, err)

			// Verify
			assert.NotEmpty(t, listedChannels)
			assert.Len(t, listedChannels, 1)

			// Cleanup
			_ = repo.DeleteNetwork(context.Background(), mockNetwork.ID)
		})
	}
}

func TestIrcRepo_CheckExistingNetwork(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewIrcRepo(log, db)
		mockNetwork := getMockIrcNetwork()

		t.Run(fmt.Sprintf("CheckExistingNetwork_NoMatch [%s]", dbType), func(t *testing.T) {
			// Execute
			existingNetwork, err := repo.CheckExistingNetwork(context.Background(), &mockNetwork)
			assert.NoError(t, err)

			// Verify
			assert.Nil(t, existingNetwork)
		})

		t.Run(fmt.Sprintf("CheckExistingNetwork_MatchFound [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreNetwork(context.Background(), &mockNetwork)
			assert.NoError(t, err)

			// Execute
			existingNetwork, err := repo.CheckExistingNetwork(context.Background(), &mockNetwork)
			assert.NoError(t, err)

			// Verify
			assert.NotNil(t, existingNetwork)
			assert.Equal(t, mockNetwork.Server, existingNetwork.Server)
			assert.Equal(t, mockNetwork.Port, existingNetwork.Port)
			assert.Equal(t, mockNetwork.Nick, existingNetwork.Nick)

			// Cleanup
			_ = repo.DeleteNetwork(context.Background(), mockNetwork.ID)
		})
	}
}

func TestIrcRepo_StoreNetworkChannels(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewIrcRepo(log, db)
		mockNetwork := getMockIrcNetwork()
		mockChannels := []domain.IrcChannel{getMockIrcChannel()}

		t.Run(fmt.Sprintf("StoreNetworkChannels_DeleteOldChannels [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreNetwork(context.Background(), &mockNetwork)
			assert.NoError(t, err)

			err = repo.StoreNetworkChannels(context.Background(), mockNetwork.ID, mockChannels)
			assert.NoError(t, err)

			// Execute
			err = repo.StoreNetworkChannels(context.Background(), mockNetwork.ID, []domain.IrcChannel{})
			assert.NoError(t, err)

			// Verify
			existingChannels, err := repo.ListChannels(mockNetwork.ID)
			assert.NoError(t, err)
			assert.Len(t, existingChannels, 0)

			// Cleanup
			_ = repo.DeleteNetwork(context.Background(), mockNetwork.ID)
		})

		t.Run(fmt.Sprintf("StoreNetworkChannels_InsertNewChannels [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreNetwork(context.Background(), &mockNetwork)
			assert.NoError(t, err)

			// Execute
			err = repo.StoreNetworkChannels(context.Background(), mockNetwork.ID, mockChannels)
			assert.NoError(t, err)

			// Verify
			existingChannels, err := repo.ListChannels(mockNetwork.ID)
			assert.NoError(t, err)
			assert.Len(t, existingChannels, len(mockChannels))

			// Cleanup
			_ = repo.DeleteNetwork(context.Background(), mockNetwork.ID)
		})
	}
}

func TestIrcRepo_UpdateChannel(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewIrcRepo(log, db)
		mockNetwork := getMockIrcNetwork()
		mockChannel := getMockIrcChannel()

		t.Run(fmt.Sprintf("UpdateChannel_Success [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreNetwork(context.Background(), &mockNetwork)
			assert.NoError(t, err)

			err = repo.StoreChannel(context.Background(), mockNetwork.ID, &mockChannel)
			assert.NoError(t, err)

			// Update mockChannel properties
			updatedChannel := mockChannel
			updatedChannel.Enabled = false
			updatedChannel.Name = "updated_name"
			updatedChannel.Password = "updated_password"

			// Execute
			err = repo.UpdateChannel(&updatedChannel)
			assert.NoError(t, err)

			// Verify
			fetchedChannels, err := repo.ListChannels(mockNetwork.ID)
			assert.NoError(t, err)

			fetchedChannel := fetchedChannels[0]
			assert.Equal(t, updatedChannel.Enabled, fetchedChannel.Enabled)
			assert.Equal(t, updatedChannel.Name, fetchedChannel.Name)
			assert.Equal(t, updatedChannel.Password, fetchedChannel.Password)

			// Cleanup
			_ = repo.DeleteNetwork(context.Background(), mockNetwork.ID)
		})
	}
}

func TestIrcRepo_UpdateInviteCommand(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewIrcRepo(log, db)
		mockNetwork := getMockIrcNetwork()

		t.Run(fmt.Sprintf("UpdateInviteCommand_Success [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreNetwork(context.Background(), &mockNetwork)
			assert.NoError(t, err)

			// Update invite_command
			newInviteCommand := "/new_invite_command"
			err = repo.UpdateInviteCommand(mockNetwork.ID, newInviteCommand)
			assert.NoError(t, err)

			// Verify
			updatedNetwork, err := repo.ListNetworks(context.Background())
			assert.NoError(t, err)

			assert.Equal(t, newInviteCommand, updatedNetwork[0].InviteCommand)

			// Cleanup
			_ = repo.DeleteNetwork(context.Background(), mockNetwork.ID)
		})
	}
}
