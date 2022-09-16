package messages

import (
	"nathejk.dk/nathejk/types"
)

/*
	type Envelope struct {
		Type string          `json:"type"`
		Body json.RawMessage `json:"body"`
	}
*/
type ChangeSet []string

func (s ChangeSet) Exists(key string) bool {
	for _, prop := range s {
		if prop == key {
			return true
		}
	}
	return false
}
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

type Nathejk_Member struct {
	Entity struct {
		ID              types.ID          `json:"id"`
		TeamID          types.LegacyID    `json:"teamId"`
		Number          string            `json:"number"`
		CreatedUts      string            `json:"createdUts"`
		PausedUts       string            `json:"pausedUts"`
		InhqUts         string            `json:"inhqUts"`
		DiscontinuedUts string            `json:"discontinuedUts"`
		DeletedUts      string            `json:"deletedUts"`
		Title           string            `json:"title"`
		Address         string            `json:"address"`
		PostalCode      string            `json:"postalCode"`
		Mail            types.Email       `json:"mail"`
		Phone           types.PhoneNumber `json:"phone"`
		ContactPhone    types.PhoneNumber `json:"contactPhone"`
		ContactSms      types.PhoneNumber `json:"contactSms"`
		SpejderTelefon  types.PhoneNumber `json:"spejderTelefon"`
		BirthDate       types.Date        `json:"birthDate"`
		Returning       string            `json:"returning"`
		Remark          string            `json:"remark"`
		PhotoID         string            `json:"photoId"`
	} `json:"entity"`
	Changed ChangeSet `json:"changed"`
	Sql     struct {
		NathejkMember string `json:"nathejk_member"`
	} `json:"sql"`
}
type Nathejk_Team struct {
	Entity struct {
		ID                   types.LegacyID     `json:"id"`
		CreatedUts           string             `json:"createdUts"`
		VerifiedUts          string             `json:"verifiedUts"`
		OpenedUts            string             `json:"openedUts"`
		FinishedUts          string             `json:"finishedUts"`
		CanceledUts          string             `json:"canceledUts"`
		DeletedUts           string             `json:"deletedUts"`
		LastModifyUts        string             `json:"lastModifyUts"`
		StartUts             string             `json:"startUts"`
		FinishUts            string             `json:"finishUts"`
		SignupStatusTypeName types.SignupStatus `json:"signupStatusTypeName"`
		TypeName             types.TeamType     `json:"typeName"`
		ParentTeamID         string             `json:"parentTeamId"`
		IP                   string             `json:"ip"`
		Title                string             `json:"title"`
		Gruppe               string             `json:"gruppe"`
		Division             string             `json:"division"`
		Korps                string             `json:"korps"`
		ContactTitle         string             `json:"contactTitle"`
		ContactAddress       string             `json:"contactAddress"`
		ContactPostalCode    string             `json:"contactPostalCode"`
		ContactMail          types.Email        `json:"contactMail"`
		ContactPhone         types.PhoneNumber  `json:"contactPhone"`
		ContactRole          string             `json:"contactRole"`
		TeamNumber           string             `json:"teamNumber"`
		LigaNumber           string             `json:"ligaNumber"`
		LigaNumberVerified   *string            `json:"ligaNumberVerified"`
		LokNumber            string             `json:"lokNumber"`
		MemberCount          string             `json:"memberCount"`
		PaidMemberCount      string             `json:"paidMemberCount"`
		Paid                 string             `json:"paid"`
		PaidStatus           *string            `json:"paidStatus"`
		Remark               string             `json:"remark"`
		PhotoID              string             `json:"photoId"`
		PhotoUts             string             `json:"photoUts"`
		AdvSpejdNightCount   string             `json:"advSpejdNightCount"`
		ArrivalName          string             `json:"arrivalName"`
		CheckedAtStart       string             `json:"checkedAtStart"`
	} `json:"entity"`
	Changed ChangeSet `json:"changed"`
	Sql     struct {
		NathejkTeam string `json:"nathejk_team"`
	} `json:"sql"`
}
