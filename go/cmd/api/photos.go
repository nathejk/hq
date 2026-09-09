package main

// Photographs of the patrols, and which one represents a team.
//
// # hq holds the metadata, foto holds the bytes
//
// The photographs are taken by the camera crew and ingested by the foto service,
// which stores the objects in a content-addressed blob store and publishes
// `NATHEJK.<year>.patrulje.<teamId>.photographed`. hq projects those events (see
// the photo table) so a patrol page can list a team's pictures with no call to
// another service — but it does not hold a single byte of image data.
//
// So every URL here points at foto's `/photos/<ref>` route. The alternative,
// proxying the bytes through hq, was rejected: the ref is the hash of the bytes, so
// foto's responses are immutable and cached forever by the browser, and a proxy
// would put an authenticated, uncacheable hop in front of that for no gain. It also
// keeps the one rule that matters in foto, where it is enforced by its read model:
// stored originals carry the upload's EXIF — possibly the location a child was
// photographed in — and are never served. hq could not honour that rule for foto
// even if it wanted to.

import (
	"log"
	"net/http"
	"strings"
	"time"

	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/photo"
)

// photoURL is where the browser fetches an object's bytes.
//
// Absolute, because it is another service's URL. A relative path would be resolved
// against hq and 404 there.
func (app *application) photoURL(ref string) string {
	if ref == "" {
		return ""
	}
	return strings.TrimSuffix(app.config.photo.baseurl, "/") + "/photos/" + ref
}

// photoRendition is one cached size, as the client sees it.
type photoRendition struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Bytes  int    `json:"bytes"`
}

// photoResponse is one photograph, with URLs instead of refs.
//
// The ref is kept as well: it is the identity of the photograph, and it is what the
// cover picker sends back. Everything else a client needs to lay the picture out
// (dimensions, and each rendition's dimensions) travels with it, so choosing a size
// never requires downloading one to measure it.
type photoResponse struct {
	Ref      string `json:"ref"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumbUrl,omitempty"`

	Width  int `json:"width"`
	Height int `json:"height"`

	Type      string `json:"type,omitempty"`
	Attention bool   `json:"attention,omitempty"`

	CapturedAt *time.Time `json:"capturedAt,omitempty"`

	// Cover marks the photograph shown wherever the team is represented by one
	// picture. Exactly one photograph in the list has it, if the list is non-empty:
	// the organizer's choice where there is one, otherwise the newest.
	Cover bool `json:"cover"`

	// Chosen distinguishes the two. Only a chosen cover can be un-chosen, and the
	// picker uses this to show whether the current cover is a decision or a default.
	Chosen bool `json:"chosen"`

	Renditions []photoRendition `json:"renditions,omitempty"`
}

// showPatruljePhotosHandler lists one patrulje's photographs, newest first.
//
// An empty list rather than a 404 for a team with no photographs: "this team has
// none" and "no such team" are different answers, and the page asking has already
// loaded the team.
func (app *application) showPatruljePhotosHandler(w http.ResponseWriter, r *http.Request) {
	teamID := app.ReadNamedParam(r, "id")
	if teamID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	year := string(app.YearSlug(r))

	photos, err := app.models.Photo.ByTeam(r.Context(), year, teamID)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	// A cover ref that no longer names one of the team's photographs is ignored
	// rather than repaired: the photograph may have been purged, and the row is still
	// the honest record of a choice somebody made. The fallback below then applies.
	coverRef, err := app.models.PhotoCover.Ref(year, teamID)
	if err != nil {
		// Losing the choice costs the ordering, not the pictures.
		log.Printf("PhotoCover.Ref %q", err)
		coverRef = ""
	}

	body := app.photoPayload(photos, coverRef)
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{
		"teamId": teamID,
		"photos": body,
	}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// photoPayload converts the read model to the client's shape and resolves the
// cover.
//
// The cover is resolved here, on the server, rather than left to each view: three
// screens show a patrol's photograph and "chosen, else newest" is one rule, not
// three implementations of it.
func (app *application) photoPayload(photos []photo.Photo, coverRef string) []photoResponse {
	out := make([]photoResponse, 0, len(photos))
	coverIndex, chosen := -1, false

	for i, p := range photos {
		item := photoResponse{
			Ref:        p.Ref,
			URL:        app.photoURL(p.Ref),
			ThumbURL:   app.photoURL(p.ThumbRef),
			Width:      p.Width,
			Height:     p.Height,
			Type:       p.Type,
			Attention:  p.Attention,
			CapturedAt: p.CapturedAt,
		}
		for _, rendition := range p.Renditions {
			item.Renditions = append(item.Renditions, photoRendition{
				Name:   rendition.Name,
				URL:    app.photoURL(rendition.Ref),
				Width:  rendition.Width,
				Height: rendition.Height,
				Bytes:  rendition.Bytes,
			})
		}
		out = append(out, item)

		if coverRef != "" && p.Ref == coverRef && coverIndex == -1 {
			coverIndex, chosen = i, true
		}
	}

	// ByTeam returns newest first, so index 0 is the default cover.
	if coverIndex == -1 && len(out) > 0 {
		coverIndex = 0
	}
	if coverIndex >= 0 {
		out[coverIndex].Cover = true
		out[coverIndex].Chosen = chosen
	}
	return out
}

// showPatruljeCoversHandler lists one photograph per team for the whole year.
//
// One request for a list of ~200 patrols. A thumbnail component that fetched its
// own team's photographs would turn one page into 200 requests, and the list only
// ever shows one picture per row.
//
// Not mounted under /patrulje/... because httprouter cannot have both a static
// segment and `:id` at the same position — `/api/patrulje/covers` would collide
// with `/api/patrulje/:id`.
func (app *application) showPatruljeCoversHandler(w http.ResponseWriter, r *http.Request) {
	covers, err := app.models.PhotoCover.Covers(r.Context(), string(app.YearSlug(r)))
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	type coverResponse struct {
		TeamID   string `json:"teamId"`
		Ref      string `json:"ref"`
		URL      string `json:"url"`
		ThumbURL string `json:"thumbUrl,omitempty"`
		Width    int    `json:"width"`
		Height   int    `json:"height"`
		Count    int    `json:"count"`
		Chosen   bool   `json:"chosen"`
	}

	out := make([]coverResponse, 0, len(covers))
	for _, c := range covers {
		out = append(out, coverResponse{
			TeamID:   c.TeamID,
			Ref:      c.Ref,
			URL:      app.photoURL(c.Ref),
			ThumbURL: app.photoURL(c.ThumbRef),
			Width:    c.Width,
			Height:   c.Height,
			Count:    c.Count,
			Chosen:   c.Chosen,
		})
	}

	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"covers": out}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// selectPatruljeCoverHandler records which photograph represents a patrulje.
//
// An empty ref clears the choice, which is not the same as having none: the page
// falls back to the newest photograph either way, but the operator can undo a
// decision they made.
func (app *application) selectPatruljeCoverHandler(w http.ResponseWriter, r *http.Request) {
	teamID := app.ReadNamedParam(r, "id")
	if teamID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	year := string(app.YearSlug(r))

	var input struct {
		Ref string `json:"ref"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	// The chosen photograph must be one of this team's. Checked here rather than in
	// the entity because only the handler has the team's photographs in front of it —
	// and without the check a typo would set a cover that resolves to nothing, which
	// looks exactly like "no photo" on every page that shows one.
	photos, err := app.models.Photo.ByTeam(r.Context(), year, teamID)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	if input.Ref != "" {
		known := false
		for _, p := range photos {
			if p.Ref == input.Ref {
				known = true
				break
			}
		}
		if !known {
			app.NotFoundResponse(w, r)
			return
		}
	}

	if err := app.commands.PhotoCover.Select(year, teamID, input.Ref); err != nil {
		log.Printf("PhotoCover.Select %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}

	// The projection applies asynchronously, so the response is built from the value
	// just accepted rather than read back — reading would race the projection and
	// usually lose, telling the operator their click did nothing. The live signal
	// replaces this with the projected truth a moment later.
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{
		"teamId": teamID,
		"photos": app.photoPayload(photos, input.Ref),
	}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
