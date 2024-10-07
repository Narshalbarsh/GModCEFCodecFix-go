package steam_steamid

// TODO this is blindly ported from the python version and is untested, there are probably bugs

import (
	"fmt"
	"regexp"
	"strconv"
)

// SteamID struct represents a Steam ID
type SteamID struct {
	Universe  int
	Type      int
	Instance  int
	AccountID uint32
}

// Constants for SteamID Universe, Type, and Instance
const (
	UniverseInvalid = iota
	UniversePublic
	UniverseBeta
	UniverseInterval
	UniverseDev
)

const (
	TypeInvalid = iota
	TypeIndividual
	TypeMultiseat
	TypeGameServer
	TypeAnonGameServer
	TypePending
	TypeContentServer
	TypeClan
	TypeChat
	TypeP2PSuperSeeder
	TypeAnonUser
)

const (
	InstanceAll = iota
	InstanceDesktop
	InstanceConsole
	InstanceWeb = 4
)

const (
	AccountIDMask       = 0xFFFFFFFF
	AccountInstanceMask = 0x000FFFFF
)

// ChatInstanceFlags maps different chat instances
var ChatInstanceFlags = map[string]int{
	"Clan":     (AccountInstanceMask + 1) >> 1,
	"Lobby":    (AccountInstanceMask + 1) >> 2,
	"MMSLobby": (AccountInstanceMask + 1) >> 3,
}

// TypeChars maps Steam ID types to corresponding characters
var TypeChars = map[int]string{
	TypeInvalid:        "I",
	TypeIndividual:     "U",
	TypeMultiseat:      "M",
	TypeGameServer:     "G",
	TypeAnonGameServer: "A",
	TypePending:        "P",
	TypeContentServer:  "C",
	TypeClan:           "g",
	TypeChat:           "T",
	TypeAnonUser:       "a",
}

// NewSteamID initializes a new SteamID based on input
func NewSteamID(input string) (*SteamID, error) {
	sid := &SteamID{
		Universe: UniverseInvalid,
		Type:     TypeInvalid,
		Instance: InstanceAll,
	}

	if input == "" {
		return sid, nil
	}

	// Regular expressions to match the input format
	reg := regexp.MustCompile(`^STEAM_([0-5]):([0-1]):([0-9]+)$`)
	reg3 := regexp.MustCompile(`^\[([a-zA-Z]):([0-5]):([0-9]+)(:[0-9]+)?\]`)
	if mat := reg.FindStringSubmatch(input); mat != nil {
		universe, _ := strconv.Atoi(mat[1])
		if universe == 0 {
			universe = UniversePublic
		}
		sid.Universe = universe
		sid.Type = TypeIndividual
		sid.Instance = InstanceDesktop
		accountID, _ := strconv.Atoi(mat[3])
		accountIdMultiplier, _ := strconv.Atoi(mat[2])
		sid.AccountID = uint32(accountID*2 + accountIdMultiplier)

	} else if mat3 := reg3.FindStringSubmatch(input); mat3 != nil {
		sid.Universe, _ = strconv.Atoi(mat3[2])

		accountID, _ := strconv.Atoi(mat3[3])
		sid.AccountID = uint32(accountID)

		typeChar := mat3[1]

		if mat3[4] != "" {
			sid.Instance, _ = strconv.Atoi(mat3[4][1:])
		} else if typeChar == "U" {
			sid.Instance = InstanceDesktop
		}

		switch typeChar {
		case "c":
			sid.Instance = ChatInstanceFlags["Clan"]
			sid.Type = TypeChat
		case "L":
			sid.Instance = ChatInstanceFlags["Lobby"]
			sid.Type = TypeChat
		default:
			sid.Type = getTypeFromChar(typeChar)
		}
	} else {
		inputVal, err := strconv.ParseUint(input, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unknown ID: %s", input)
		}

		// Bitwise operations to extract account ID, instance, type, and universe
		sid.AccountID = uint32(inputVal & AccountIDMask)
		sid.Instance = int((inputVal >> 32) & 0xFFFFF)
		sid.Type = int((inputVal >> 52) & 0xF)
		sid.Universe = int((inputVal >> 56) & 0xFF)
	}

	return sid, nil
}

// getTypeFromChar retrieves SteamID Type from the character representation
func getTypeFromChar(typeChar string) int {
	for typ, char := range TypeChars {
		if char == typeChar {
			return typ
		}
	}
	return TypeInvalid
}

// Steam2 renders Steam2 ID
func (sid *SteamID) Steam2(newerFormat bool) (string, error) {
	if sid.Type != TypeIndividual {
		return "", fmt.Errorf("can't get Steam2 ID for non-individual ID")
	}
	universe := sid.Universe
	if !newerFormat && universe == UniversePublic {
		universe = 0
	}
	return fmt.Sprintf("STEAM_%d:%d:%d", universe, sid.AccountID&1, sid.AccountID/2), nil
}

// Steam3 renders Steam3 ID
func (sid *SteamID) Steam3() string {
	typeChar := TypeChars[sid.Type]
	if typeChar == "" {
		typeChar = "i"
	}

	if sid.Instance&ChatInstanceFlags["Clan"] != 0 {
		typeChar = "c"
	} else if sid.Instance&ChatInstanceFlags["Lobby"] != 0 {
		typeChar = "L"
	}

	renderInstance := (sid.Type == TypeAnonGameServer || sid.Type == TypeMultiseat ||
		(sid.Type == TypeIndividual && sid.Instance != InstanceDesktop))

	if renderInstance {
		return fmt.Sprintf("[%s:%d:%d:%d]", typeChar, sid.Universe, sid.AccountID, sid.Instance)
	}
	return fmt.Sprintf("[%s:%d:%d]", typeChar, sid.Universe, sid.AccountID)
}

// IsValid validates the SteamID
func (sid *SteamID) IsValid() bool {
	if sid.Type <= TypeInvalid || sid.Type > TypeAnonUser {
		return false
	}
	if sid.Universe <= UniverseInvalid || sid.Universe > UniverseDev {
		return false
	}
	if sid.Type == TypeIndividual && (sid.AccountID == 0 || sid.Instance > InstanceWeb) {
		return false
	}
	if sid.Type == TypeClan && (sid.AccountID == 0 || sid.Instance != InstanceAll) {
		return false
	}
	if sid.Type == TypeGameServer && sid.AccountID == 0 {
		return false
	}
	return true
}
