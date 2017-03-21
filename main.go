// Server application for the futa-world text adventure engine
// Currently supports telnet connections and also TODO: provides a web interface.
// Please configure the variables in config.json for your server.
// The game world is read from game.json.
package main

// TODO: do I need to use an ampersand like &SomeStruct{} if I want to make a new one without using a newSomeStruct() function?

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/firstrow/tcp_server"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"
)

const engineVer = "0.0.1"

// String displayed on connect
// TODO: separate these into game.json
// TODO: separate the "please send any messagen to continue" to separate string
const welcomeString = "\r\nWelcome to futa.world! This text adventure game was created by deltaryz.\r\n\r\nWARNING: EXPLICIT CONTENT\r\n"
const introMessage = "You are PLACEHOLDER_NAME, a young mare from Ponyville. You have awoken to find yourself in an unknown location, and all you know is that you are REALLY itching to fuck something with your massive futa schlong.\r\n\r\n"

// locker utility to prevent map collision
var plock sync.Mutex
var tlock sync.Mutex

// Player map
var Players = make(map[string]*Player)
var tcpConnections = make(map[*tcp_server.Client]string)

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
	Name    string   `json:"name"`    // Username given by the Player
	Inv     ItemList `json:"inv"`     // Array of Item objects currently owned by the Player
	Pos     pos      `json:"pos"`     // Room position in x/y coordinates on the map
	Game    Game     `json:"game"`    // Player's unique game json
	Health  int64    `json:"health"`  // take a wild guess
	Arousal int64    `json:"arousal"` // she's a kinky fucker
}

// Converts a `pos` type to a string with the format "XxY"
func posToString(inputPos *pos) string {
	return strconv.Itoa(inputPos.X) + "x" + strconv.Itoa(inputPos.Y)
}

// Converts a string formatted "XxY" to a `pos` object
func stringToPos(inputPos string) (*pos, error) {
	separatedStrings := strings.Split(inputPos, "x")
	x, errX := strconv.Atoi(separatedStrings[0])
	y, errY := strconv.Atoi(separatedStrings[1])

	if errX != nil {
		return nil, errX
	}

	if errY != nil {
		return nil, errY
	}

	return &pos{X: x, Y: y}, nil
}

// Stats returns a string with the Player's stats, used for game start & stats command
func (p *Player) Stats() string {
	info := ""

	info += "Health: " + strconv.FormatInt(p.Health, 10) + "\r\nArousal: " + strconv.FormatInt(p.Arousal, 10) // TODO: remove Arousal, add logic to properly display the wildcard stat from game.json

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
		Name:    username,
		Inv:     ItemList{newDildo()},
		Pos:     pos{X: 0, Y: 0},
		Health:  10,
		Arousal: 10, // TODO: change to a "wildcard" stat, game.json can define its name & this name is only used for string output
	}
	return result
}

type Game struct {
	GameTitle      string           `json:"game_title"`      // The title of the game
	StartInventory ItemList         `json:"start_inventory"` // The inventory the player starts with
	StartRoom      string           `json:"start_room"`      // Coordinates in "XxY" format
	Rooms          map[string]*Room `json:"rooms"`           // map of position coordinates in "XxY" format to room objects TODO: init game property of players upon acct creation
}

// Slice of Items, such as an inventory or chest
type ItemList []*Item

// Position struct, for easy .pos.X
type pos struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// generic Item struct, inherited by all Items
type Item struct {
	Name       string   `json:"name"`               // Name of the Item
	AltNames   []string `json:"alt_names"`          // Alternate shorthand names the player may refer to the item by
	Desc       string   `json:"desc"`               // Short description of the Item
	Labels     []string `json:"labels"`             // Array of descriptive labels which apply to the Item (think booru tagging)
	Weight     float32  `json:"weight"`             // Weight of the Item
	Owned      bool     `json:"owned"`              // Is the Item currently in the Player's inventory?
	Location   string   `json:"location,omitempty"` // States where in the room the Item is (ground, wall, "the pedestal", etc) - unused if the Item is currently owned
	Contents   ItemList `json:"contents"`           // so i herd u liek items / Used for chests/containers to contain more items. Yes, this works recursively.
	Capacity   int      `json:"capacity"`           // Amount of items that can be contained within - set to 0 if it is not a container
	Obtainable bool     `json:"obtainable"`         // Whether or not the player can pick up this item
}

// Pick up an Item from the room
func (i *Item) pickUp() string {
	if !i.Owned {
		// TODO: pick up Items
		return fmt.Sprintf("You pick up the %s.", i.Name)
	} else {
		return "You already have the " + i.Name + "!"
	}
}

// Drop an Item on the ground
func (i *Item) drop() string {
	if i.Owned {
		// TODO: drop Items on ground
		return fmt.Sprintf("You drop the %s onto the ground.", i.Name)
	} else {
		return "You can't drop that, you aren't holding it!"
	}
}

// Set the Item to an error Item
func (i *Item) makeError() error {
	i.Labels = []string{"err"}
	i.Location = ""
	i.Owned = false
	i.Desc = "Something went wrong with Item generation. Please contact the developer."
	i.Name = "Error Item"
	i.Weight = 696969

	return nil
}

// Returns the name and description of the Item
func (i *Item) getBasicInfo() []string {
	return []string{i.Name, i.Desc}
}

/*
// Instantiates a new empty Item.
// Please ensure that you initialize it with appropriate values before using.
func newEmptyItem() *Item {
	return &Item{} // TODO: change this to a func to construct an item from the world json
}
*/

// Initializes an Item with Dildo properties
// TODO: remove this, replace with default items field in game.json
func newDildo() *Item {
	result := &Item{
		Name:       "Modest Dildo",
		AltNames:   []string{"dildo"},
		Desc:       "A medium sized, unassuming dildo. It is purple.",
		Labels:     []string{"sextoy", "blunt", "weapon"}, // TODO: keep database of labels and their usage ingame
		Weight:     2,
		Owned:      true,
		Obtainable: true,
	}
	return result
}

// generic Room struct, is used to create each room
type Room struct {
	Pos   pos      `json:"pos"`   // Position in the world of the room.
	Name  string   `json:"name"`  // Name of the room.
	Desc  string   `json:"desc"`  // Description of the room, output of "look" command
	Items ItemList `json:"items"` // Items contained in the room.
	Exits Exits    `json:"exits"` // Valid exits of the room
}

// Which exits of a room are valid exits?
// This can be changed at runtime (unlocking doors)
type Exits struct {
	North bool `json:"north"`
	South bool `json:"south"`
	East  bool `json:"east"`
	West  bool `json:"west"`
}

// create a new room based on given coordinates from the given world json
func newRoom(coords pos) *Room { // TODO: receive json in argument
	room := &Room{
		Pos:   coords,
		Name:  "",         // TODO: grab name from world json
		Desc:  "",         // TODO: grab desc from world json
		Items: ItemList{}, // TODO: grab items from world json
		Exits: Exits{},    // TODO: grab exits from world json
	}

	return room
}

type Config struct {
	TelnetEnabled bool `json:"telnet_enabled"`
	TelnetPort    int  `json:"telnet_port"`
	WebEnabled    bool `json:"web_enabled"`
	WebPort       int  `json:"web_port"`
}

// Main function
func main() {

	telnetPort := 23
	webPort := 80

	// Command line flags for loading alternate config and game files
	configPath := flag.String("config", "config.json", "-config <path>")
	gamePath := flag.String("game", "game.json", "-game <path>")

	flag.Parse()

	// Attempt to open these files
	configFile, configErr := ioutil.ReadFile(*configPath)
	gameFile, gameErr := ioutil.ReadFile(*gamePath)

	if configErr != nil {
		fmt.Println("Error reading config file!\r\n" + configErr.Error())
	}

	if gameErr != nil {
		fmt.Println("Error reading world file!\r\n" + gameErr.Error())
	}

	if configErr != nil || gameErr != nil {
		os.Exit(69)
	}

	var configSettings Config
	errConfig := json.Unmarshal(configFile, &configSettings)

	var masterGame Game
	errGame := json.Unmarshal(gameFile, &masterGame)

	telnetPort = configSettings.TelnetPort
	webPort = configSettings.WebPort
	//fmt.Println(configSettings.TelnetPort)

	if errConfig != nil {
		fmt.Println("Error reading config file, expect possible derpage when trying to connect!\r\n")
	}

	if errGame != nil {
		fmt.Println("Error reading game file, expect many things to explode!\r\n")
	}

	// TODO: load master map json into variable

	// TCP setup
	server := tcp_server.New("localhost:" + strconv.FormatInt(int64(telnetPort), 10))

	// new TCP client has connected
	server.OnNewClient(func(c *tcp_server.Client) {
		// new client connected
		// lets send some dank ass message
		c.Send("This text adventure game is running on the futa-world engine: https://github.com/techniponi/futa-world\r\n") // You can remove this if you really want, but I ask kindly that you do not.
		c.Send(welcomeString)
		c.Send("Please type any message and press enter to continue.\r\n")

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
					c.Send("Thank you for playing futa.world!\r\n\r\n") // TODO: have this use global name string
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

	fmt.Println("Web server at port " + strconv.Itoa(webPort) + " would be running right now, if it was implemented yet.")
	// TODO: web interface
}

// Generic message handler function
// Designed to be client/protocol-independent for maximum portability
// Returns a response string to print to the client, the string username of the client, and a bool stating whether a TCPMap needs to be created (only used for TCP).
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
			p2, playerExistsFromArg := getPlayer(args[1])
			if playerExists {
				response += fmt.Sprintf("You are already logged in as player: %s!", p.Name)
			} else {
				response += "Initializing game...\r\n"
				username = args[1] // username is not set because Player hasn't logged in yet
				needsTcpMap = true // in case the client is tcp, unused otherwise
				// Player already exists
				if playerExistsFromArg {
					response += "User exists, attempting to log in...\r\n\r\n"
					fmt.Println("User " + p2.Name + " succesfully logged in")
					response += fmt.Sprintf("%s%s", introMessage, p2.Stats()) // TODO: don't use the intro string here, also describe room
				} else { // create new account
					response += "Username does not exist, creating new player profile...\r\n\r\n"
					setPlayer(username, newPlayer(username)) // set Player in database
					tmpPlayer, ok2 := getPlayer(username)    // get Player back from database to ensure successful creation
					if ok2 {
						fmt.Println("User " + tmpPlayer.Name + " succesfully created account")

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

	// observe surroundings
	case "look":
		// TODO: look command
		break

	default:
		// TODO: custom actions (in game.json, this should be VERY extensive so that users don't need to add source code for most math comparisons/simple item manipulation)
		if playerExists {
			response += "Error: invalid command"
		}
		break
	}

	return response, username, needsTcpMap

}
