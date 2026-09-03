// The map backdrops, in one place (PRD 010, PRD 011).
//
// # Why a factory rather than a shared object
//
// A Leaflet tile layer instance belongs to **one** map: adding the same instance to a second map
// detaches it from the first, and the symptom is a map that mysteriously goes blank when a dialog
// opens. `KortView` originally held these as a module-level object of instances, which was fine while
// it was the only map in the app. The track dialog (task 150) is a second one, so this returns fresh
// instances per caller.
//
// The definitions themselves are worth sharing: an operator switching between the checkpoint map and a
// patrol's track should be looking at the same ground rendered the same way, and the Dataforsyningen
// token and its terms-of-use attribution should exist once.

import L from 'leaflet'

/** The layer shown first, and the one operators reason about the terrain with. */
export const DEFAULT_BASE_LAYER = 'Topografisk 1:25.000'

/**
 * Fresh base layers for one map.
 *
 * Order matters: `L.control.layers` lists them in insertion order, and the topographic sheet is what
 * the printed maps are drawn on, so it leads.
 */
export function createBaseLayers(): Record<string, L.TileLayer | L.TileLayer.WMS> {
  return {
    [DEFAULT_BASE_LAYER]: L.tileLayer.wms('https://api.dataforsyningen.dk/dtk_25_DAF', {
      layers: 'DTK25',
      format: 'image/png',
      transparent: true,
      attribution:
        '&copy; <a target="_blank" href="https://download.kortforsyningen.dk/content/vilk%C3%A5r-og-betingelser">Styrelsen for Dataforsyning og Effektivisering</a>',
      // @ts-ignore – extra param passed as query string by Leaflet
      token: '0d5816d7e175e934301f0277686c43f8',
      maxZoom: 19,
    } as L.WMSOptions),
    Luftfoto: L.tileLayer(
      'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
      {
        attribution: '&copy; Esri &mdash; Sources: Esri, DigitalGlobe, Earthstar Geographics',
        maxZoom: 19,
      },
    ),
    OpenStreetMap: L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
      maxZoom: 19,
    }),
    Topografisk: L.tileLayer('https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png', {
      attribution:
        '&copy; <a href="https://opentopomap.org">OpenTopoMap</a> (<a href="https://creativecommons.org/licenses/by-sa/3.0/">CC-BY-SA</a>)',
      maxZoom: 17,
    }),
  }
}

/**
 * Roughly the race area, for a map with nothing to fit yet.
 *
 * Only used until real data arrives and `fitBounds` takes over — a map centred on the null island
 * while a track loads looks broken.
 */
export const RACE_AREA_CENTER: [number, number] = [55.7, 12.1]
export const RACE_AREA_ZOOM = 11
