import { describe, it, expect, vi } from 'vitest'
import { nextTick, ref } from 'vue'
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
})
