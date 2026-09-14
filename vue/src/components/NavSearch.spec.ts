// @vitest-environment jsdom
//
// The navigation search box (PRD 014 §7, task 185).
//
// Mounted rather than logic-tested because everything here *is* interaction: a document-level
// shortcut, a focus call, and a form submit. The two behaviours worth pinning are the ones that would
// be actively harmful if wrong — a shortcut that fires while somebody is writing an SOS note, and a
// shortcut that swallows a browser key for a user who never wanted our search.

import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { createRouter, createMemoryHistory, type Router } from 'vue-router'
import NavSearch from './NavSearch.vue'

const makeRouter = (): Router =>
  createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'home', component: { template: '<div />' } },
      { path: '/search', name: 'search', component: { template: '<div />' } },
    ],
  })

let router: Router
let wrapper: VueWrapper

const open = async (query: Record<string, string> = {}) => {
  router = makeRouter()
  await router.push({ path: '/', query })
  await router.isReady()
  // attachTo the document: the shortcut listens on `document`, so a detached tree could not be
  // focused by it and the test would pass for the wrong reason.
  wrapper = mount(NavSearch, { global: { plugins: [router] }, attachTo: document.body })
  return wrapper
}

const input = () => wrapper.find('input#nav-person-search').element as HTMLInputElement

const pressSlash = (target: EventTarget = document.body) => {
  const event = new KeyboardEvent('keydown', { key: '/', cancelable: true, bubbles: true })
  target.dispatchEvent(event)
  return event
}

beforeEach(() => {
  document.body.innerHTML = ''
})

afterEach(() => {
  wrapper?.unmount()
})

describe('the box', () => {
  it('is a search landmark with a label, reachable without a mouse', async () => {
    await open()
    expect(wrapper.find('form[role="search"]').exists()).toBe(true)
    expect(wrapper.find('label[for="nav-person-search"]').text()).toBe('Søg efter person')
    expect(input().getAttribute('aria-label')).toContain('Tryk /')
  })

  it('submits to /search?q=… on Enter alone', async () => {
    await open()
    await wrapper.find('input').setValue('20309696')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(router.currentRoute.value.name).toBe('search')
    expect(router.currentRoute.value.query.q).toBe('20309696')
  })

  it('trims, and goes nowhere on an empty query', async () => {
    await open()
    await wrapper.find('input').setValue('  Koch  ')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(router.currentRoute.value.query.q).toBe('Koch')

    await wrapper.find('input').setValue('   ')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    // Still on the previous search, not pushed to an empty one.
    expect(router.currentRoute.value.query.q).toBe('Koch')
  })

  it('shows what is being searched when arriving from a link', async () => {
    await open({ q: 'Koch' })
    expect(input().value).toBe('Koch')
  })

  it('is not a results dropdown', async () => {
    // Out of scope by decision (task 185): it would compete with SearchView for the same job.
    await open({ q: 'Koch' })
    expect(wrapper.findAll('li')).toHaveLength(0)
    expect(wrapper.text()).not.toContain('match')
  })
})

describe('the shortcut', () => {
  it('focuses the box from anywhere on the page', async () => {
    await open()
    expect(document.activeElement).not.toBe(input())

    const event = pressSlash()
    expect(document.activeElement).toBe(input())
    expect(event.defaultPrevented).toBe(true)
  })

  it('does not fire from inside another input', async () => {
    await open()
    const other = document.createElement('input')
    document.body.appendChild(other)
    other.focus()

    const event = pressSlash(other)
    expect(document.activeElement).toBe(other)
    // And the slash must still reach the field the operator was typing into.
    expect(event.defaultPrevented).toBe(false)
  })

  it('does not fire from inside a textarea or a rich-text editor', async () => {
    await open()

    const textarea = document.createElement('textarea')
    document.body.appendChild(textarea)
    textarea.focus()
    expect(pressSlash(textarea).defaultPrevented).toBe(false)
    expect(document.activeElement).toBe(textarea)

    // The mail view's Quill editor is a contenteditable div, not an input.
    const editor = document.createElement('div')
    editor.setAttribute('contenteditable', 'true')
    document.body.appendChild(editor)
    expect(pressSlash(editor).defaultPrevented).toBe(false)
  })

  it('leaves a modified slash to the browser', async () => {
    await open()
    for (const modifier of ['ctrlKey', 'metaKey', 'altKey'] as const) {
      const event = new KeyboardEvent('keydown', {
        key: '/',
        cancelable: true,
        bubbles: true,
        [modifier]: true,
      })
      document.body.dispatchEvent(event)
      expect(event.defaultPrevented).toBe(false)
    }
  })

  it('stops listening once it is gone', async () => {
    await open()
    wrapper.unmount()
    // No focus target left; the assertion is that this does not throw.
    expect(() => pressSlash()).not.toThrow()
  })
})
