import { describe, it, expect, vi } from 'vitest'
import { nextTick, computed, ref } from 'vue'
import { useDeferredApply } from './useDeferredApply'

// The pattern this covers protects unsaved work from live updates, so the cases
// that matter are the awkward ones: a payload arriving mid-edit, several arriving
// mid-edit, and the edit ending.

describe('useDeferredApply', () => {
  it('applies an already-cached value immediately, so a warm view does not wait', () => {
    const source = ref<string | undefined>('cached')
    const apply = vi.fn()

    useDeferredApply(source, ref(false), apply)

    expect(apply).toHaveBeenCalledWith('cached')
  })

  it('ignores undefined: nothing loaded is not something to apply', async () => {
    const source = ref<string | undefined>(undefined)
    const apply = vi.fn()

    useDeferredApply(source, ref(false), apply)
    await nextTick()

    expect(apply).not.toHaveBeenCalled()
  })

  it('applies payloads that arrive while idle', async () => {
    const source = ref<string | undefined>(undefined)
    const apply = vi.fn()
    useDeferredApply(source, ref(false), apply)

    source.value = 'first'
    await nextTick()

    expect(apply).toHaveBeenCalledWith('first')
  })

  it('holds a payload back while paused, and reports that it is waiting', async () => {
    const source = ref<string | undefined>('initial')
    const paused = ref(false)
    const apply = vi.fn()
    const { updatesWaiting } = useDeferredApply(source, paused, apply)
    apply.mockClear()

    paused.value = true
    source.value = 'while editing'
    await nextTick()

    expect(apply).not.toHaveBeenCalled()
    expect(updatesWaiting.value).toBe(true)
  })

  it('applies what was held back once the edit ends', async () => {
    const source = ref<string | undefined>('initial')
    const paused = ref(false)
    const apply = vi.fn()
    const { updatesWaiting } = useDeferredApply(source, paused, apply)
    apply.mockClear()

    paused.value = true
    source.value = 'while editing'
    await nextTick()
    paused.value = false
    await nextTick()

    expect(apply).toHaveBeenCalledWith('while editing')
    expect(updatesWaiting.value).toBe(false)
  })

  // Several revalidations can land during one long edit. Replaying them in turn
  // would apply superseded data and, worse, could leave the newest unapplied.
  it('applies only the newest of several deferred payloads', async () => {
    const source = ref<string | undefined>('initial')
    const paused = ref(true)
    const apply = vi.fn()
    useDeferredApply(source, paused, apply)
    apply.mockClear()

    source.value = 'second'
    await nextTick()
    source.value = 'third'
    await nextTick()
    paused.value = false
    await nextTick()

    expect(apply).toHaveBeenCalledTimes(1)
    expect(apply).toHaveBeenCalledWith('third')
  })

  // The operator opened a dialog and closed it again without anything arriving.
  // Re-applying here would be a pointless rebuild of the tree, and on a page whose
  // rows carry drag state that is not free.
  it('does not apply on unpause when nothing arrived', async () => {
    const source = ref<string | undefined>('initial')
    const paused = ref(false)
    const apply = vi.fn()
    const { updatesWaiting } = useDeferredApply(source, paused, apply)
    apply.mockClear()

    paused.value = true
    await nextTick()
    paused.value = false
    await nextTick()

    expect(apply).not.toHaveBeenCalled()
    expect(updatesWaiting.value).toBe(false)
  })

  it('defers again on a second edit', async () => {
    const source = ref<string | undefined>('initial')
    const paused = ref(false)
    const apply = vi.fn()
    const { updatesWaiting } = useDeferredApply(source, paused, apply)

    paused.value = true
    source.value = 'a'
    await nextTick()
    paused.value = false
    await nextTick()
    expect(apply).toHaveBeenLastCalledWith('a')

    paused.value = true
    source.value = 'b'
    await nextTick()
    expect(updatesWaiting.value).toBe(true)
    expect(apply).toHaveBeenLastCalledWith('a')

    paused.value = false
    await nextTick()
    expect(apply).toHaveBeenLastCalledWith('b')
  })

  // A pause that begins before anything is cached must not strand the first
  // payload: the operator could open a dialog on an empty page and never see data.
  it('applies the first payload when the pause ends, even if it arrived before any', async () => {
    const source = ref<string | undefined>(undefined)
    const paused = ref(true)
    const apply = vi.fn()
    useDeferredApply(source, paused, apply)

    source.value = 'first'
    await nextTick()
    expect(apply).not.toHaveBeenCalled()

    paused.value = false
    await nextTick()
    expect(apply).toHaveBeenCalledWith('first')
  })

  // The trap that deadlocked KortSettingsDialog, kept here because it is a property of *composing*
  // this helper rather than of the helper itself, and it is not visible from either side alone.
  //
  // The pause condition is usually "the buffers differ from the source" — a dirty check. Such a
  // condition is **true by construction** the moment a *different* source value arrives, because the
  // buffers still hold the old one. Pausing on it means the new value is never applied, which keeps
  // the buffers unequal, which keeps the pause on: a deadlock with no unpausing event, presenting as
  // an editor stuck showing the wrong record and refusing to move because of changes nobody made.
  //
  // So a dirty-derived pause must be scoped to the identity the buffers hold — defer a *refresh* of
  // the loaded record, never the arrival of another one.
  describe('when the pause is derived from the source (a dirty check)', () => {
    interface Row {
      id: string
      name: string
    }

    /** Mimics the dialog: buffers, a dirty check against the source, and the identity guard. */
    const harness = () => {
      const source = ref<Row | undefined>(undefined)
      const buffer = ref({ name: '' })
      const loadedId = ref<string | undefined>(undefined)

      const dirty = computed(() => !!source.value && buffer.value.name !== source.value.name)
      const deferSameRow = computed(() => dirty.value && source.value?.id === loadedId.value)

      const apply = vi.fn((row: Row) => {
        buffer.value = { name: row.name }
        loadedId.value = row.id
      })

      useDeferredApply(source, deferSameRow, apply)
      return { source, buffer, dirty, apply }
    }

    it('applies a newly selected row even though the stale buffer makes it look dirty', async () => {
      const { source, buffer, dirty, apply } = harness()

      source.value = { id: 'a', name: 'Kort 1' }
      await nextTick()

      expect(apply).toHaveBeenCalledTimes(1)
      expect(buffer.value.name).toBe('Kort 1')
      // And nothing is reported unsaved, so a guard built on this cannot refuse the next click.
      expect(dirty.value).toBe(false)
    })

    it('switches rows without demanding a save, when nothing was edited', async () => {
      const { source, buffer, dirty, apply } = harness()

      source.value = { id: 'a', name: 'Kort 1' }
      await nextTick()
      source.value = { id: 'b', name: 'Kort 2' }
      await nextTick()

      expect(apply).toHaveBeenCalledTimes(2)
      expect(buffer.value.name).toBe('Kort 2')
      expect(dirty.value).toBe(false)
    })

    // The protection the guard exists for must survive the fix: a refresh of the row being edited is
    // still held back, so a live payload cannot overwrite the field under the cursor.
    it('still defers a refresh of the row currently being edited', async () => {
      const { source, buffer, apply } = harness()

      source.value = { id: 'a', name: 'Kort 1' }
      await nextTick()

      buffer.value = { name: 'Kort 1 — mid-edit' }
      source.value = { id: 'a', name: 'Renamed by somebody else' }
      await nextTick()

      expect(apply).toHaveBeenCalledTimes(1)
      expect(buffer.value.name).toBe('Kort 1 — mid-edit')
    })
  })
})
