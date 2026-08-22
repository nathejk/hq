# 094 — Multi-select acceptance in the arrivals queue

**Status:** open
**Priority:** medium
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

## Description

A car arrives with three scouts and they arrive together (PRD 007 §6). The *I bil — på vej
hertil* section supports selecting several rows and pressing **Modtaget** once, with one
placering applied to the whole selection — scouts off one car are normally bedded down
together.

One HTTP request per member. Unlike whole-team collection (which is deliberately one atomic
request, because three calls could split a patrol across two states), there is no atomicity
requirement here: each acceptance is an independent fact about one child, and a partly
failed batch leaves the rest visibly still in the queue rather than in a wrong state. Report
what failed, by name, and leave those rows selected.

The action bar appears only while something is selected, so the section looks the same as it
does today when nobody is working in it.

## Acceptance Criteria

- [ ] Multi-select on the *I bil* table only
- [ ] One **Modtaget** action over the selection, asking once for a placering
- [ ] One request per member; a partial failure names the members that failed and leaves
      them selected
- [ ] Selection clears for the members that succeeded
- [ ] Action bar hidden when nothing is selected

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
