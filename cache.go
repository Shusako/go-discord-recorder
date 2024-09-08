package main

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
)

// holds the mapping from SSRC to UserID
type SSRCMap struct {
	mutex sync.Mutex
	// something is weird, VoiceSpeakingUpdate.SSRC is an int but the SSRC field in an opus packet is a uint32, so we'll just use a string
	mapData map[string]string
}

var ssrcMapOnce sync.Once
var _ssrcMap *SSRCMap

// GetSSRCMap provides a global, singleton access to SSRCMap
func GetSSRCMap() *SSRCMap {
	ssrcMapOnce.Do(func() {
		_ssrcMap = &SSRCMap{
			mapData: make(map[string]string),
		}
	})
	return _ssrcMap
}

func (m *SSRCMap) Add(ssrc int, userId string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.mapData[string(ssrc)] = userId
}

func (m *SSRCMap) GetUserId(ssrc int) (string, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	userId, exists := m.mapData[string(ssrc)]
	return userId, exists
}

func (m *SSRCMap) GetUser(ssrc uint32) (discordgo.User, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	userId, exists := m.mapData[string(ssrc)]
	if !exists {
		return discordgo.User{}, false
	}

	user, exists := GetUserCache().GetUserLazy(userId)
	return user, exists
}

//=============================================================================

type UserCache struct {
	mutex   sync.RWMutex
	mapData map[string]discordgo.User
}

var userCacheOnce sync.Once
var _userCache *UserCache

// GetUserCache provides a global, singleton access to UserCache
func GetUserCache() *UserCache {
	userCacheOnce.Do(func() {
		_userCache = &UserCache{
			mapData: make(map[string]discordgo.User),
		}
	})
	return _userCache
}

func (m *UserCache) Add(userId string, user discordgo.User) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.mapData[userId] = user
}

func (m *UserCache) getUserFromCache(userId string) (discordgo.User, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	user, exists := m.mapData[userId]
	return user, exists
}

func (m *UserCache) GetUserLazy(userId string) (user discordgo.User, ok bool) {
	cachedUser, exists := m.getUserFromCache(userId)
	if exists {
		return cachedUser, true
	}

	liveUser, err := Context.discordSession.User(userId)
	if err != nil {
		fmt.Println("error fetching user,", err)
		return discordgo.User{}, false
	}

	m.Add(userId, *liveUser)
	return *liveUser, true
}

func (m *UserCache) GetUsernameOrDefault(userId string, defaultReturn string) string {
	user, exists := m.GetUserLazy(userId)
	if !exists {
		return defaultReturn
	}
	return user.Username
}

//=============================================================================

func registerCacheHandlers(discordSession *discordgo.Session, voiceConnection *discordgo.VoiceConnection) {
	// This specific handler attaches to the voice connection and lets us map SSRC -> UserID
	voiceConnection.AddHandler(voiceSpeakingUpdateHandler)
}

func voiceSpeakingUpdateHandler(vc *discordgo.VoiceConnection, event *discordgo.VoiceSpeakingUpdate) {
	GetSSRCMap().Add(event.SSRC, event.UserID)
}
