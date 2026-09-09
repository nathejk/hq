import { describe, expect, it } from 'vitest'
import { photoUrlFor, type PatruljePhoto } from './patruljePhotos'

// Only the pure part is tested here: the two resources are thin useLiveResource
// wrappers, and what is actually worth pinning is the size choice — getting it wrong
// is a 500 KB download behind a 64px thumbnail, which nothing would ever report.

const photo = (renditions: PatruljePhoto['renditions']): Pick<PatruljePhoto, 'url' | 'renditions'> => ({
  url: 'https://foto/photos/display',
  renditions,
})

describe('photoUrlFor', () => {
  it('picks the smallest rendition at least as wide as the target', () => {
    const p = photo([
      { name: 'thumb256', url: 'u256', width: 256, height: 192, bytes: 1 },
      { name: 'thumb1024', url: 'u1024', width: 1024, height: 768, bytes: 2 },
    ])
    expect(photoUrlFor(p, 200)).toBe('u256')
    expect(photoUrlFor(p, 256)).toBe('u256')
    expect(photoUrlFor(p, 257)).toBe('u1024')
  })

  it('falls back to the display image when no rendition is big enough', () => {
    const p = photo([{ name: 'thumb256', url: 'u256', width: 256, height: 192, bytes: 1 }])
    expect(photoUrlFor(p, 1400)).toBe('https://foto/photos/display')
  })

  // A photograph recorded before renditions existed has none. Consumers must degrade
  // to the display image rather than treat it as a broken record.
  it('falls back when there are no renditions at all', () => {
    expect(photoUrlFor(photo(undefined), 64)).toBe('https://foto/photos/display')
    expect(photoUrlFor(photo([]), 64)).toBe('https://foto/photos/display')
  })
})
