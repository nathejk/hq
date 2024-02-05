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

type YearSlug Slug

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
type DepartmentID ID

func NewDepartmentID() DepartmentID {
	return DepartmentID("dep-" + uuid.New().String())
}

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
	MemberStatusWaiting   MemberStatus = "waiting"
	MemberStatusTransit   MemberStatus = "transit"
	MemberStatusEmergency MemberStatus = "emergency"
	MemberStatusHQ        MemberStatus = "hq"
	MemberStatusOut       MemberStatus = "out"
)

type PingType string

const (
	PingTypeSignup        PingType = "signup"
	PingTypeMobilepayLink PingType = "mobilepay"
)

type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
