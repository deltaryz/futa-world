// Server application for http://futa.world, an erotic pony-themed text adventure.
// Currently supports telnet connections and also TODO: provide a web interface.
package main

// TODO: do I need to use an ampersand like &SomeStruct{} if I want to make a new one without using a newSomeStruct() function?

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

const introMessage = "You are PLACEHOLDER_NAME, a young mare from Ponyville. You have awoken to find yourself in an unknown location, and all you know is that you are REALLY itching to fuck something with your massive futa schlong.\r\n\r\n"

// init data types

// locker utility to prevent map collision
var plock sync.Mutex
var tlock sync.Mutex

// Player map
var Players = make(map[string]*Player)
var tcpConnections = make(map[*tcp_server.Client]string)

// GENERAL HELPER FUNCTIONS

// Gets a Player from the map, makes sure to wait until nothing else is accessing it
func getPlayer(username string) (*Player, bool) {
	plock.Lock()
	defer plock.Unlock()

	pl, exists := Players[username]

	return pl, exists
}

// Sets a Player in the map, ignoring whether it exists or not
func setPlayer(username string, p *Player) bool {
	plock.Lock()
	defer plock.Unlock()

	Players[username] = p

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
type Player struct {
	name    string        `json:"name"`    // Username given by the Player
	inv     ItemList      `json:"inv"`     // Array of Item objects currently owned by the Player
	pos     pos           `json:"pos"`     // Room position in x/y coordinates on the map
	world   map[pos]*Room `json:"world"`   // map of position structs to room objects TODO: init world property of players upon acct creation
	health  int64         `json:"health"`  // take a wild guess
	arousal int64         `json:"arousal"` // she's a kinky fucker
}

// Stats returns a string with the Player's stats, used for game start & stats command
func (p *Player) Stats() string {
	info := ""

	info += "Health: " + strconv.FormatInt(p.health, 10) + "\r\nArousal: " + strconv.FormatInt(p.arousal, 10)

	return info
}

// Inventory returns a string describing the contents of the Player's inventory
func (p *Player) Inventory() string {
	info := ""
	// TODO: finish inventory method
	return info
}

// Create new Player with beginning-game attributes
func newPlayer(username string) *Player {
	result := &Player{
		name:    username,
		inv:     ItemList{newDildo()},
		pos:     pos{x: 0, y: 0},
		health:  10,
		arousal: 10,
	}
	return result
}

// Slice of Items, such as an inventory or chest
type ItemList []*Item

// Position struct, for easy .pos.X
type pos struct {
	x int `json:"x"`
	y int `json:"y"`
}

// generic Item struct, inherited by all Items
type Item struct {
	name     string   `json:"name"`               // Name of the Item
	desc     string   `json:"desc"`               // Short description of the Item
	labels   []string `json:"labels"`             // Array of descriptive labels which apply to the Item (think booru tagging)
	weight   float32  `json:"weight"`             // Weight of the Item
	owned    bool     `json:"owned"`              // Is the Item currently in the Player's inventory?
	location string   `json:"location,omitempty"` // States where in the room the Item is (ground, wall, "the pedestal", etc) - unused if the Item is currently owned
	contents ItemList `json:"contents"`           // so i herd u liek items / Used for chests/containers to contain more items
}

// Pick up an Item from the room
func (i *Item) pickUp() string {
	if !i.owned {
		// TODO: pick up Items
		return fmt.Sprintf("You pick up the %s.", i.name)
	} else {
		return "You already have the " + i.name + "!"
	}
}

// Drop an Item on the ground
func (i *Item) drop() string {
	if i.owned {
		// TODO: drop Items on ground
		return fmt.Sprintf("You drop the %s.", i.name)
	} else {
		return "You can't drop that, you aren't holding it!"
	}
}

// Set the Item to an error Item
func (i *Item) makeError() error {
	i.labels = []string{"err"}
	i.location = ""
	i.owned = false
	i.desc = "Something went wrong with Item generation. Please contact the developer."
	i.name = "Error Item"
	i.weight = 696969

	return nil
}

// Returns the name and description of the Item
func (i *Item) getBasicInfo() []string {
	return []string{i.name, i.desc}
}

// Instantiates a new empty Item.
// Please ensure that you initialize it with appropriate values before using.
func newEmptyItem() *Item {
	return &Item{}
}

// Initializes an Item with Dildo properties
func newDildo() *Item {
	result := &Item{
		desc: "A medium sized, unassuming dildo. It is purple.",
		name: "Modest Dildo",
	}
	return result
}

// generic Room struct, is used to create each room
type Room struct {
	pos   pos      `json:"pos"`   // Position in the world of the room.
	name  string   `json:"name"`  // Name of the room.
	desc  string   `json:"desc"`  // Description of the room, output of "look" command
	items ItemList `json:"items"` // Items contained in the room.
	exits Exits    `json:"exits"` // Valid exits of the room
}

// Which exits of a room are valid exits?
// This can be changed at runtime (unlocking doors)
type Exits struct {
	north bool `json:"north"`
	south bool `json:"south"`
	east  bool `json:"east"`
	west  bool `json:"west"`
}

// create a new room based on given coordinates from the given world json
func newRoom(coords pos) *Room { // TODO: receive json in argument
	room := &Room{
		pos:   coords,
		name:  "",         // TODO: grab name from world json
		desc:  "",         // TODO: grab desc from world json
		items: ItemList{}, // TODO: grab items from world json
		exits: Exits{},    // TODO: grab exits from world json
	}

	return room
}

// Main function
func main() {

	// TODO: load master map json into variable

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
				if args[0] == "quit" || args[0] == "exit" {
					c.Send("Thank you for playing futa.world!\r\n\r\n")
					c.Close()
				} else { // user is not quitting game

					// pass command to the generic interpreter
					msg, uname, needsMap := messageReceived(args, username)

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
func messageReceived(args []string, username string) (string, string, bool) {

	// start with empty string
	response := ""
	// only used for tcp connections, indicates whether a tcp mapping needs to be made
	needsTcpMap := false

	p, playerExists := getPlayer(username)

	// which command did the Player use?
	// just fyi this switch is gonna be FUCKING LONG
	switch args[0] {

	// User is attempting to log in
	case "login":
		if len(args) > 1 {
			// TODO: sanitize login string
			if playerExists {
				response += "You are already logged in!"
			} else {
				response += "Initializing game...\r\n"
				username = args[1] // username is not set because Player hasn't logged in yet
				needsTcpMap = true // in case the client is tcp, unused otherwise
				// Player already exists
				if playerExists {
					response += "User exists, attempting to log in...\r\n\r\n"
					fmt.Println("User " + p.name + " succesfully logged in")
					response += fmt.Sprintf("%s%s", introMessage, p.Stats()) // TODO: don't use the intro string here, also describe room
				} else { // create new account
					response += "Username does not exist, creating new Player profile...\r\n\r\n"
					setPlayer(username, newPlayer(username)) // set Player in database
					tmpPlayer, ok2 := getPlayer(username)    // get Player back from database to ensure successful creation
					if ok2 {
						fmt.Println("User " + tmpPlayer.name + " succesfully created account")

						// TODO: create world from json, store in player object

						response += fmt.Sprintf("%s%s", introMessage, tmpPlayer.Stats()) // TODO: describe current room
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

	// display the Player's stats
	case "stats":
		if playerExists {
			response += p.Stats()
		}
		break

	// list contents of inventory
	case "inv":
	case "inventory":
		if playerExists {
			response += p.Inventory()
		}
		break

	default:
		if playerExists {
			response += "Error: invalid command"
		}
		break
	}

	return response, username, needsTcpMap

}
