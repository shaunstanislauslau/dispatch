package server

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/viper"

	"github.com/khlieng/dispatch/storage"
)

type connectDefaults struct {
	Name        string
	Host        string
	Port        int
	Channels    []string
	Password    bool
	SSL         bool
	ReadOnly    bool
	ShowDetails bool
}

type indexData struct {
	Defaults connectDefaults
	Servers  []Server
	Channels []*storage.Channel

	// Users in the selected channel
	Users *Userlist

	// Last messages in the selected channel
	Messages *Messages
}

func getIndexData(r *http.Request, state *State) *indexData {
	data := indexData{}

	data.Defaults = connectDefaults{
		Name:        viper.GetString("defaults.name"),
		Host:        viper.GetString("defaults.host"),
		Port:        viper.GetInt("defaults.port"),
		Channels:    viper.GetStringSlice("defaults.channels"),
		Password:    viper.GetString("defaults.password") != "",
		SSL:         viper.GetBool("defaults.ssl"),
		ReadOnly:    viper.GetBool("defaults.readonly"),
		ShowDetails: viper.GetBool("defaults.show_details"),
	}

	if state == nil {
		return &data
	}

	servers, err := state.user.GetServers()
	if err != nil {
		return nil
	}
	connections := state.getConnectionStates()
	for _, server := range servers {
		server.Password = ""
		server.Username = ""
		server.Realname = ""

		data.Servers = append(data.Servers, Server{
			Server: server,
			Status: newConnectionUpdate(server.Host, connections[server.Host]),
		})
	}

	channels, err := state.user.GetChannels()
	if err != nil {
		return nil
	}
	for i, channel := range channels {
		channels[i].Topic = channelStore.GetTopic(channel.Server, channel.Name)
	}
	data.Channels = channels

	server, channel := getTabFromPath(r.URL.EscapedPath())
	if isInChannel(channels, server, channel) {
		data.addUsersAndMessages(server, channel, state)
		return &data
	}

	server, channel = parseTabCookie(r, r.URL.Path)
	if isInChannel(channels, server, channel) {
		data.addUsersAndMessages(server, channel, state)
	}

	return &data
}

func (d *indexData) addUsersAndMessages(server, channel string, state *State) {
	users := channelStore.GetUsers(server, channel)
	if len(users) > 0 {
		d.Users = &Userlist{
			Server:  server,
			Channel: channel,
			Users:   users,
		}
	}

	messages, hasMore, err := state.user.GetLastMessages(server, channel, 50)
	if err == nil && len(messages) > 0 {
		m := Messages{
			Server:   server,
			To:       channel,
			Messages: messages,
		}

		if hasMore {
			m.Next = messages[0].ID
		}

		d.Messages = &m
	}
}

func isInChannel(channels []*storage.Channel, server, channel string) bool {
	if channel != "" {
		for _, ch := range channels {
			if server == ch.Server && channel == ch.Name {
				return true
			}
		}
	}
	return false
}

func getTabFromPath(rawPath string) (string, string) {
	path := strings.Split(strings.Trim(rawPath, "/"), "/")
	if len(path) == 2 {
		name, err := url.PathUnescape(path[1])
		if err == nil && isChannel(name) {
			return path[0], name
		}
	}
	return "", ""
}

func parseTabCookie(r *http.Request, path string) (string, string) {
	if path == "/" {
		cookie, err := r.Cookie("tab")
		if err == nil {
			v, err := url.PathUnescape(cookie.Value)
			if err == nil {
				tab := strings.SplitN(v, ";", 2)

				if len(tab) == 2 && isChannel(tab[1]) {
					return tab[0], tab[1]
				}
			}
		}
	}
	return "", ""
}
