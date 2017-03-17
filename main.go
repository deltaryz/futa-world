// Server application for http://futa.world, an erotic pony-themed text adventure.
// Currently supports telnet connections and also provides a web interface.
package main

import (
	"fmt"
	"github.com/firstrow/tcp_server"
	"strconv"
	"strings"
	"sync"
)

// metadata constants
const ver = "0.0.1"
const telnetPort = 23
const webPort = 80

// String displayed on connect
const welcomeString = "\r\nWelcome to futa.world! This is version " + ver + ", created by deltaryz.\r\n\r\nWARNING: EXPLICIT CONTENT\r\nYou must be at least 18 years of age to play this game. If you agree that you are 18 or older, please type any message and press enter.\r\n"

// init data types

// locker utility to prevent map collision
var plock sync.Mutex
var tlock sync.Mutex

// player map
var players = make(map[string]*player)
var tcpConnections = make(map[*tcp_server.Client]string)

// GENERAL HELPER FUNCTIONS

// Gets a player from the map, makes sure to wait until nothing else is accessing it
func getPlayer(username string) (*player, bool) {
	plock.Lock()
	defer plock.Unlock()

	pl, exists := players[username]

	return pl, exists
}

// Sets a player in the map, ignoring whether it exists or not
func setPlayer(username string, p *player) bool {
	plock.Lock()
	defer plock.Unlock()

	players[username] = p

	return true
}

// Gets a TCP client's associated username
func getTCPPlayer(conn *tcp_server.Client) (string, bool) {
	tlock.Lock()
	defer tlock.Unlock()

	username, exists := tcpConnections[conn]

	return username, exists
}

// Sets a TCP client's associated username
func setTCPPlayer(conn *tcp_server.Client, username string) bool {
	tlock.Lock()
	defer tlock.Unlock()

	tcpConnections[conn] = username
	return true
}

// Player object
type player struct {
	name    string   // Username given by the player
	inv     itemList // Array of item objects currently owned by the player
	pos     pos      // Room position in x/y coordinates on the map
	health  int64    // take a wild guess
	arousal int64    // she's a kinky fucker
}

// Stats returns a string with the player's stats
func (p *player) Stats() string {
	info := ""

	info += "Health: " + strconv.FormatInt(p.health, 10) + "\r\nArousal: " + strconv.FormatInt(p.arousal, 10)

	return info
}

// Create new player with beginning-game attributes
func newPlayer(username string) *player {
	result := &player{
		name:    username,
		inv:     itemList{newDildo()},
		pos:     pos{x: 0, y: 0},
		health:  10,
		arousal: 0,
	}
	return result
}

// Slice of items, such as an inventory or chest
type itemList []*item

// Position struct, for easy .pos.X
type pos struct {
	x int
	y int
}

// generic item struct, inherited by all items
type item struct {
	name     string   // Name of the item
	desc     string   // Short description of the item
	labels   []string // Array of descriptive labels which apply to the item (think booru tagging)
	weight   float32  // Weight of the item
	owned    bool     // Is the item currently in the player's inventory?
	location string   // States where in the room the item is (ground, wall, "the pedestal", etc) - unused if the item is currently owned
}

// Pick up an item from the room
func (i *item) pickUp() string {
	if !i.owned {
		// TODO: pick up items
		return fmt.Sprintf("You pick up the %s.", i.name)
	} else {
		return "You already have the " + i.name + "!"
	}
}

// Drop an item on the ground
func (i *item) drop() string {
	if i.owned {
		// TODO: drop items on ground
		return fmt.Sprintf("You drop the %s.", i.name)
	} else {
		return "You can't drop that, you aren't holding it!"
	}
}

// Set the item to an error item
func (i *item) makeError() error {
	i.labels = []string{"err"}
	i.location = ""
	i.owned = false
	i.desc = "Something went wrong with item generation. Please contact the developer."
	i.name = "Error Item"
	i.weight = 696969

	return nil
}

// Returns the name and description of the item
func (i *item) getBasicInfo() []string {
	return []string{i.name, i.desc}
}

// Instantiates a new item.
// Please ensure that you initialize it with an item type.
func newEmptyItem() *item {
	return &item{}
}

// Initializes an item with Dildo properties
func newDildo() *item {
	result := &item{
		desc: "A medium sized, unassuming dildo. It is purple.",
		name: "Modest Dildo",
	}
	return result
}

// Main function
func main() {

	// TODO: save/load maps from file, automatic saving

	// TCP setup
	server := tcp_server.New("localhost:" + strconv.FormatInt(telnetPort, 10))

	// new TCP client has connected
	server.OnNewClient(func(c *tcp_server.Client) {
		// new client connected
		// lets send some dank ass message
		c.Send(welcomeString)

		fmt.Println(c.Conn().LocalAddr())
	})

	// TCP client has sent a new message
	server.OnNewMessage(func(c *tcp_server.Client, message string) {

		// split string
		args := strings.Fields(message)

		if len(args) <= 0 {
			c.Send("\r\n\r\n>")
		} else {

			// TODO: proper intro message
			// check if user is logged in before doing anything
			username, exists := getTCPPlayer(c) // "username" is unusable if the connection has not logged in

			// give login message if the user is not logged in
			if !exists && args[0] != "login" {
				c.Send("\r\nSend \"quit\" to exit.\r\nPlease use the \"login\" command to start or resume your session.\r\nUsage: login <username>")
			}

			// make sure user actually sent something
			if len(args) > 0 {

				// user is quitting game
				if args[0] == "quit" {
					c.Send("Thank you for playing futa.world!\r\n\r\n")
					c.Close()
				} else { // user is not quitting game

					// pass command to the generic interpreter
					msg, uname, needsMap := messageReceived(args, username, exists)

					// if we need to map the connection to a username
					if needsMap {
						setTCPPlayer(c, uname)
					}

					// send to client, add newlines for readability
					c.Send(msg + "\r\n\r\n>")
				}

			}
		}
	})

	// TCP client has disconnected
	server.OnClientConnectionClosed(func(c *tcp_server.Client, err error) {
		// connection with client lost
		// TODO: clean up character data, save their shit
	})

	// start TCP server
	server.Listen()

	// TODO: web interface
}

// Generic message handler function
// Designed to be client/protocol-independent for maximum portability
func messageReceived(args []string, username string, playerIsSet bool) (string, string, bool) {

	// start with empty string
	response := ""
	// only used for tcp connections, indicates whether a tcp mapping needs to be made
	needsTcpMap := false

	switch args[0] {

	// User is attempting to log in
	case "login":
		if len(args) > 1 {
			// TODO: sanitize login string
			if playerIsSet {
				response += "You are already logged in!"
			} else {
				response += "Initializing game...\r\n"
				username = args[1] // username is not set because player hasn't logged in yet
				needsTcpMap = true // in case the client is tcp, unused otherwise
				pl, existCheck := getPlayer(username)
				// player already exists
				if existCheck {
					// TODO: passwords
					response += "User exists, attempting to log in...\r\n\r\n"
					fmt.Println("User " + pl.name + " succesfully logged in")
					response += "You are a pony. aaaaa finish this later\r\n\r\n" + pl.Stats() // TODO: finish new game string
				} else { // create new account
					response += "Username does not exist, creating new player profile...\r\n\r\n"
					ok := setPlayer(username, newPlayer(username))
					tmpPlayer, ok := getPlayer(username)
					if ok {
						fmt.Println("User " + tmpPlayer.name + " succesfully created account")
						response += "You are a pony. aaaaa finish this later\r\n\r\n" + pl.Stats() // TODO: finish new game string
					} else {
						fmt.Println("Account creation derped")
						response += "An error occured with logging in."
						needsTcpMap = false
					}
				}
			}
		} else {
			response += "Error: not enough arguments"
		}
		break

	default:
		if playerIsSet {
			response += "Error: invalid command"
		}
		break
	}

	return response, username, needsTcpMap

}
