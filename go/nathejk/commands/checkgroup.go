package commands

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/requestctx"
	"nathejk.dk/nathejk/table/checkgroup"
	"nathejk.dk/nathejk/table/checkpoint"
	"nathejk.dk/superfluids/streaminterface"
)

type querier interface {
	GetByID(context.Context, types.CheckgroupID) (*checkgroup.Checkgroup, error)
	//GetCheckpointByID(context.Context, types.CheckpointID) (*checkpoint.Checkpoint, error)
}
type CheckgroupQuerier struct {
	Checkgroup interface {
		GetByID(context.Context, types.CheckgroupID) (*checkgroup.Checkgroup, error)
	}
	Checkpoint interface {
		GetByID(context.Context, types.CheckpointID) (*checkpoint.Checkpoint, error)
	}
}

type checkgroupcommand struct {
	p streaminterface.Publisher
	q querier

	producerSlug string
	yearSlug     types.YearSlug
}

func NewCheckgroup(p streaminterface.Publisher, q querier) *checkgroupcommand {
	return &checkgroupcommand{
		p: p,
		q: q,

		producerSlug: "hq-api",
		yearSlug:     "2026",
	}
}

func (cmd checkgroupcommand) Create(ctx context.Context, cg Checkgroup) (types.CheckgroupID, error) {
	return "", nil
	/*
		return cmd.Update(&UpdateRequest{
			ID:            types.ControlGroupID("ctrlgrp-" + uuid.New().String()),
			CreateRequest: *req.(*CreateRequest),
		})*/
}

type Checkgroup struct {
	Name                 *string
	ShowOnMap            *bool
	Mandatory            *bool
	Scheme               *types.CheckgroupScheme
	RelativeCheckgroupID *types.CheckgroupID
}
type Checkpoint struct {
	ID           types.CheckpointID
	Name         *string
	OpenFrom     *time.Time
	OpenUntil    *time.Time
	OpenDuration *time.Duration
	Position     *types.Position
	Address      *string
	Description  *string
	Scanners     []CheckpointScanner
}
type CheckpointScanner struct {
	UserID        types.UserID
	ScanningFrom  time.Time
	ScanningUntil time.Time
}

func (cmd checkgroupcommand) Update(ctx context.Context, ID types.CheckgroupID, cg Checkgroup) error {
	if ID == "" {
		return errors.New("Can't update controlgroup, no ID specified")
	}
	ctx = requestctx.WithValue(context.Background(), &requestctx.Value{Year: cmd.yearSlug})
	if err := cmd.UpdateCheckgroup(ctx, ID, cg); err != nil {
		return err
	}
	/*
		cps := []Checkpoint{}
		for i, cp := range cps {
			if err := cmd.UpdateCheckpoint(ctx, ID, cp, i); err != nil {
				return err
			}
		}*/
	return nil
}

func (cmd checkgroupcommand) UpdateCheckgroup(ctx context.Context, ID types.CheckgroupID, cg Checkgroup) error {
	if ID == "" {
		return errors.New("Can't update controlgroup, no ID specified")
	}
	v, ok := requestctx.ValueFrom(ctx)
	if !ok {
		return errors.New("context values missing")
	}

	cge, _ := cmd.q.GetByID(ctx, ID)
	dirty := (cge == nil) ||
		((cg.Name != nil) && (*cg.Name != cge.Name)) ||
		((cg.ShowOnMap != nil) && (*cg.ShowOnMap != cge.ShowOnMap)) ||
		((cg.Mandatory != nil) && (*cg.Mandatory != cge.Mandatory)) ||
		((cg.Scheme != nil) && (*cg.Scheme != cge.Scheme)) ||
		((cg.RelativeCheckgroupID != nil) && (*cg.RelativeCheckgroupID != *cge.RelativeCheckgroupID))

	if !dirty {
		return nil
	}
	body := messages.NathejkCheckgroupUpdated{
		CheckgroupID:         ID,
		Name:                 cg.Name,
		ShowOnMap:            cg.ShowOnMap,
		Mandatory:            cg.Mandatory,
		Scheme:               cg.Scheme,
		RelativeCheckgroupID: cg.RelativeCheckgroupID,
	}
	msg := cmd.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.checkgroup.%s.updated", v.Year, ID)))
	msg.SetBody(&body)
	msg.SetMeta(&messages.Metadata{Producer: cmd.producerSlug})

	return cmd.p.Publish(msg)
}

/*
func (cmd checkgroupcommand) UpdateCheckpoint(ctx context.Context, ID types.CheckgroupID, cp Checkpoint, sortOrder int) error {
	v, ok := requestctx.ValueFrom(ctx)
	if !ok {
		return errors.New("context values missing")
	}

	cpe, _ := cmd.q.GetByID(ctx, cp.ID)
	dirty := (cpe == nil) ||
		((cp.Name != nil) && (*cp.Name != cpe.Name)) ||
		//((cp.ShowOnMap != nil) && (*cg.ShowOnMap != cge.ShowOnMap)) ||
		((cp.OpenDuration != nil) && (*cp.OpenDuration != cpe.OpenDuration)) ||
		((cp.Address != nil) && (*cp.Address != cpe.Address)) ||
		((cp.Description != nil) && (*cp.Description != cpe.Description))

	if !dirty {
		return nil
	}
	body := messages.NathejkCheckpointUpdated{
		Name:                 cp.Name,
		RelativeTimeDuration: cp.OpenDuration,
		Address:              cp.Address,
		Remark:               cp.Description,
	}
	if (cp.OpenFrom != nil) && (cp.OpenUntil != nil) {
		body.FixedTimeRange = &types.TimeRange{
			Start: *cp.OpenFrom,
			End:   *cp.OpenUntil,
		}
	}
	msg := cmd.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.checkpoint.%s.updated", v.Year, ID)))
	msg.SetBody(&body)
	msg.SetMeta(&messages.Metadata{Producer: cmd.producerSlug})

	return cmd.p.Publish(msg)
	//for _, cp := range cps {
	/*
		checkpoint := messages.NathejkCheckgroup_Checkpoint{
			Name:     cp.Name,
			Scanners: []messages.NathejkCheckpoint_Scanner{},
		}
		switch cg.Scheme {
		case types.CheckgroupSchemeFixed:
			if (cp.OpenFrom != nil) && (cp.OpenUntil != nil) {
				checkpoint.FixedTimeRange = &types.TimeRange{
					Start: *cp.OpenFrom,
					End:   *cp.OpenUntil,
				}
			}
		case types.CheckgroupSchemeRelative:
			checkpoint.RelativeTimeDuration = cp.OpenDuration
		}
		/*
			for _, s := range cp.Scanners {
				checkpoint.Scanners = append(checkpoint.Scanners, messages.NathejkCheckpoint_Scanner{
					UserID: s.UserID,
					TimeRange: types.TimeRange{
						Start: s.ScanningFrom,
						End:   s.ScanningUntil,
					},
				})
			}*/
/*
	//body.Checkpoints = append(body.Checkpoints, checkpoint)
	msg := cmd.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.checkgroup.%s.updated", cmd.yearSlug, ID)))
	//msg := eventstream.NewMessage()
	//msg.Msg().Type = "controlgroup.updated"
	msg.SetBody(body)
	meta := messages.Metadata{Producer: cmd.producerSlug}
	//meta.RequestHeaders.Set(r.Header)
	msg.SetMeta(&meta)
	return cmd.p.Publish(msg)
*/
//}
//	return nil
//}
func (cmd checkgroupcommand) Sort(ctx context.Context, cgs []types.CheckgroupID) error {
	return nil
}
func (cmd checkgroupcommand) Delete(ctx context.Context, ID types.CheckgroupID) error {
	body := messages.NathejkCheckgroupDeleted{
		CheckgroupID: ID,
	}
	msg := cmd.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.checkgroup.%s.deleted", cmd.yearSlug, ID)))
	msg.SetBody(body)
	msg.SetMeta(&messages.Metadata{Producer: cmd.producerSlug})
	return cmd.p.Publish(msg)
}

type Date string

func (d Date) ToTime() time.Time {
	loc, err := time.LoadLocation("Europe/Copenhagen")
	if err != nil {
		log.Printf("Recoverable error %q", err)
		return time.Time{}
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", string(d), loc)
	if err != nil {
		return time.Time{}
	}
	return t
}

type CreateRequest struct {
	Name     string `json:"name"`
	Controls []struct {
		Name                 string               `json:"name"`
		Scheme               string               `json:"scheme"`
		RelativeCheckgroupID types.ControlGroupID `json:"relativeControlGroupId"`
		DateRange            struct {
			StartDate time.Time `json:"startDate"`
			EndDate   time.Time `json:"endDate"`
		} `json:"dateRange"`
		Minus    int `json:"minus"`
		Plus     int `json:"plus"`
		Scanners []struct {
			DateRange struct {
				StartDate time.Time `json:"startDate"`
				EndDate   time.Time `json:"endDate"`
			} `json:"dateRange"`
			UserID types.UserID `json:"userId"`
		} `json:"scanners"`
	} `json:"controls"`
}
type ReadRequest struct {
}
type UpdateRequest struct {
	ID types.ControlGroupID `json:"controlGroupId"`

	CreateRequest
}

type DeleteRequest struct {
	ID types.ControlGroupID `json:"controlGroupId"`
}
