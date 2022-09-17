package types

import (
	"crypto/md5"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type ID = string
type Slug = string
type Enum = string

/** Dictionary
 * Squad:  a small group of people having a particular task.
 * Patrol: a unit of six to eight Scouts or Guides forming part of a troop.
 */

type TeamID ID

func (t TeamID) New() TeamID {
	return TeamID("team-" + uuid.New().String())
}

type TeamIDs []TeamID

func (i TeamIDs) Add(ID TeamID) {
	i = append(i, ID)
}

func UniqueTeamID(teamIDs []TeamID) []TeamID {
	keys := make(map[TeamID]bool)
	list := []TeamID{}
	for _, entry := range teamIDs {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func DiffTeamID(slice1 TeamIDs, slices ...TeamIDs) TeamIDs {
	diff := TeamIDs{}

	// Loop two times, first to find slice1 strings not in slice2,
	// second loop to find slice2 strings not in slice1
	//	for i := 0; i < 2; i++ {
	for _, id := range slice1 {
		found := false
		for _, slice := range slices {
			if slice.Exists(id) {
				found = true
				break
			}
		}
		// String not found. We add it to return slice
		if !found {
			diff = append(diff, id)
		}
	}

	return diff
}

func (IDs TeamIDs) Exists(key TeamID) bool {
	for _, prop := range IDs {
		if prop == key {
			return true
		}
	}
	return false
}

/*
 func (s ChangeSet) Any(keys ...string) bool {
     for _, prop := range s {
         for _, key := range keys {
             if prop == key {
                 return true
             }
         }
     }
     return false
 }
*/

type MemberID ID

func (ID MemberID) New() MemberID {
	return MemberID("member-" + uuid.New().String())
}

type ScanID ID
type AttachmentID ID
type LegacyID ID

func (ID LegacyID) Checksum() string {
	// PHP: return substr(md5($this->id . '**@'), -5);
	md5sum := md5.Sum([]byte(string(ID) + "**@"))
	hash := hex.EncodeToString(md5sum[:])
	return hash[len(hash)-5:]
}
func (ID LegacyID) Year() uint {
	number, err := strconv.ParseUint(string(ID[0:4]), 10, 32)
	if err != nil {
		return 0
	}
	return uint(number)
}

type ControlGroupID ID
type SosID ID
type SosCommentID ID
type QrID ID

type UserID ID

func (id UserID) IsSlackUser() bool {
	return strings.HasPrefix(string(id), "slack-")
}

type Email string

type MemberStatus Enum

func (m MemberStatus) Valid() bool {
	return map[MemberStatus]bool{
		MemberStatusActive:    true,
		MemberStatusWaiting:   true,
		MemberStatusTransit:   true,
		MemberStatusEmergency: true,
		MemberStatusHQ:        true,
		MemberStatusOut:       true,
	}[m]
}

const (
	MemberStatusActive    MemberStatus = "active"
	MemberStatusWaiting                = "waiting"
	MemberStatusTransit                = "transit"
	MemberStatusEmergency              = "emergency"
	MemberStatusHQ                     = "hq"
	MemberStatusOut                    = "out"
)

type PingType string

const (
	PingTypeSignup        PingType = "signup"
	PingTypeMobilepayLink PingType = "mobilepay"
)
