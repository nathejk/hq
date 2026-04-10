<script setup>
import { nextTick, ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { PrimeIcons } from '@primevue/core/api'
import { http } from '@/plugins/axios'
import { useToast } from 'primevue/usetoast'
import Quill from 'quill'

// --- Register custom Variable blot ---
const Embed = Quill.import('blots/embed')

class VariableBlot extends Embed {
  static blotName = 'variable'
  static tagName = 'span'
  static className = 'ql-variable-token'

  static create(value) {
    const node = super.create()
    node.setAttribute('data-variable', value)
    node.setAttribute('contenteditable', 'false')
    node.setAttribute('spellcheck', 'false')
    node.textContent = value.replace(/^#+|#+$/g, '')
    return node
  }

  static value(node) {
    return node.getAttribute('data-variable')
  }

  length() {
    return 1
  }
}

Quill.register(VariableBlot)

const route = useRoute()
const router = useRouter()
const toast = useToast()

// --- Page detection ---
const currentPage = computed(() => {
  const page = route.params.page
  if (page === 'templates') return 'templates'
  if (page === 'outbox') return 'outbox'
  return 'compose'
})

const pageTitle = computed(() => {
  switch (currentPage.value) {
    case 'templates':
      return 'Skabeloner'
    case 'outbox':
      return 'Udbakke'
    default:
      return 'Skriv ny'
  }
})

// --- Recipient type ---
const recipientTypes = ref([
  { label: 'Person (Gøgler)', value: 'person', icon: 'pi pi-user' },
  { label: 'Patrulje', value: 'patrulje', icon: 'pi pi-users' },
  { label: 'Klan', value: 'klan', icon: 'pi pi-flag' }
])
const selectedRecipientType = ref('person')

const recipientTypeLabel = (value) => {
  const rt = recipientTypes.value.find((t) => t.value === value)
  return rt ? rt.label : value
}

const recipientTypeIcon = (value) => {
  const rt = recipientTypes.value.find((t) => t.value === value)
  return rt ? rt.icon : 'pi pi-envelope'
}

// --- Recipient scope (for patrulje/klan) ---
const recipientScopes = ref([
  { label: 'Kun kontaktperson', value: 'contact' },
  { label: 'Alle medlemmer', value: 'all' }
])
const selectedRecipientScope = ref('contact')

const showScopeSelector = computed(() => selectedRecipientType.value === 'patrulje' || selectedRecipientType.value === 'klan')

// --- Data sources ---
const personnel = ref([])
const patruljer = ref([])
const klaner = ref([])
const loading = ref(false)

const loadPersonnel = async () => {
  try {
    const response = await http.get('/badut')
    personnel.value = (response.data.personnel || [])
      .filter((p) => p.paidAmount > 0)
      .map((p) => ({
        id: p.id,
        name: p.name,
        email: p.email || '',
        group: p.group || '',
        korps: p.korps || ''
      }))
  } catch (error) {
    console.error('Failed to load personnel', error)
  }
}

const loadPatruljer = async () => {
  try {
    const response = await http.get('/patrulje')
    patruljer.value = (response.data.teams || [])
      .filter((p) => p.name !== '')
      .map((p) => ({
        id: p.teamId,
        name: p.name,
        teamNumber: p.teamNumber,
        group: p.group || '',
        korps: p.korps || '',
        memberCount: p.memberCount || 0,
        signupStatus: p.signupStatus
      }))
  } catch (error) {
    console.error('Failed to load patruljer', error)
  }
}

const loadKlaner = async () => {
  try {
    const response = await http.get('/lok')
    // Klaner come from the lok endpoint — flatten teams from all LOKs
    const allTeams = (response.data.teams || []).filter((k) => k.paidAmount > 0)
    klaner.value = allTeams.map((k) => ({
      id: k.id,
      name: k.name,
      group: k.group || '',
      memberCount: k.memberCount || 0
    }))
  } catch (error) {
    console.error('Failed to load klaner', error)
  }
}

onMounted(async () => {
  loading.value = true
  await Promise.all([loadPersonnel(), loadPatruljer(), loadKlaner()])
  loading.value = false
})

// --- Recipients ---
const selectedPersons = ref([])
const selectedPatruljer = ref([])
const selectedKlaner = ref([])

const selectedRecipients = computed(() => {
  switch (selectedRecipientType.value) {
    case 'person':
      return selectedPersons.value
    case 'patrulje':
      return selectedPatruljer.value
    case 'klan':
      return selectedKlaner.value
    default:
      return []
  }
})

const recipientSummary = computed(() => {
  const items = selectedRecipients.value
  if (!items || items.length === 0) return 'Ingen modtagere valgt'
  const names = items.map((r) => r.name)
  if (names.length <= 3) return names.join(', ')
  return `${names.slice(0, 3).join(', ')} (+${names.length - 3} mere)`
})

// Reset selections when switching type
watch(selectedRecipientType, () => {
  selectedRecipientScope.value = 'contact'
})

// --- Template variables ---
const variablesByType = {
  person: [
    { label: 'Navn', tag: '#NAVN#', description: 'Personens fulde navn' },
    { label: 'Gruppe', tag: '#GRUPPE#', description: 'Gruppe / division' },
    { label: 'Korps', tag: '#KORPS#', description: 'Personens korps' }
  ],
  patrulje: [
    { label: 'Holdnavn', tag: '#HOLDNAVN#', description: 'Patruljens navn' },
    { label: 'Holdnummer', tag: '#HOLDNUMMER#', description: 'Patruljens nummer' },
    { label: 'Gruppe', tag: '#GRUPPE#', description: 'Gruppe / division' },
    { label: 'Korps', tag: '#KORPS#', description: 'Patruljens korps' },
    { label: 'Antal spejdere', tag: '#ANTAL_SPEJDERE#', description: 'Antal spejdere på holdet' },
    { label: 'Kontaktperson', tag: '#KONTAKTPERSON#', description: 'Navn på kontaktperson' }
  ],
  klan: [
    { label: 'Klannavn', tag: '#KLANNAVN#', description: 'Klanens navn' },
    { label: 'Gruppe', tag: '#GRUPPE#', description: 'Gruppe / division' },
    { label: 'Antal medlemmer', tag: '#ANTAL_MEDLEMMER#', description: 'Antal medlemmer i klanen' },
    { label: 'Kontaktperson', tag: '#KONTAKTPERSON#', description: 'Navn på kontaktperson' }
  ]
}

const availableVariables = computed(() => {
  const type = editingTemplate.value ? editingTemplate.value.recipientType : selectedRecipientType.value
  return variablesByType[type] || []
})

const editorRef = ref(null)
const subjectRef = ref(null)
const lastFocusedField = ref('body') // 'subject' or 'body'
const subjectCursorPos = ref(null)

const onSubjectFocus = () => {
  lastFocusedField.value = 'subject'
}

const onSubjectBlur = (event) => {
  subjectCursorPos.value = event.target.selectionStart
}

const onEditorFocus = () => {
  lastFocusedField.value = 'body'
}

const insertVariableIntoSubject = (tag, subjectModel) => {
  const pos = subjectCursorPos.value ?? subjectModel.length
  const current = subjectModel || ''
  const before = current.slice(0, pos)
  const after = current.slice(pos)
  const needsSpaceBefore = before.length > 0 && before[before.length - 1] !== ' '
  const needsSpaceAfter = after.length > 0 && after[0] !== ' '
  const insertion = (needsSpaceBefore ? ' ' : '') + tag + (needsSpaceAfter ? ' ' : '')
  const newValue = before + insertion + after
  const newPos = pos + insertion.length
  subjectCursorPos.value = newPos
  nextTick(() => {
    const inputEl = subjectRef.value?.$el?.querySelector('input') || subjectRef.value?.$el
    if (inputEl && inputEl.focus) {
      inputEl.focus()
      inputEl.setSelectionRange(newPos, newPos)
    }
  })
  return newValue
}

const insertVariableIntoBody = (tag) => {
  const editorEl = editorRef.value
  if (editorEl && editorEl.quill) {
    const quill = editorEl.quill
    const range = quill.getSelection(true)
    const textBefore = quill.getText(Math.max(0, range.index - 1), 1)
    if (range.index > 0 && textBefore && textBefore !== ' ' && textBefore !== '\n') {
      quill.insertText(range.index, ' ')
      quill.insertEmbed(range.index + 1, 'variable', tag)
      quill.insertText(range.index + 2, ' ')
      quill.setSelection(range.index + 3)
    } else {
      quill.insertEmbed(range.index, 'variable', tag)
      quill.insertText(range.index + 1, ' ')
      quill.setSelection(range.index + 2)
    }
  } else if (editingTemplate.value) {
    editingTemplate.value.body = (editingTemplate.value.body || '') + tag
  } else {
    mail.value.body = (mail.value.body || '') + tag
  }
}

const insertVariable = (tag) => {
  if (lastFocusedField.value === 'subject') {
    if (editingTemplate.value) {
      editingTemplate.value.subject = insertVariableIntoSubject(tag, editingTemplate.value.subject)
    } else {
      mail.value.subject = insertVariableIntoSubject(tag, mail.value.subject)
    }
  } else {
    insertVariableIntoBody(tag)
  }
}

// Convert HTML with variable blots back to plain variable tags for the server
const serializeBody = (html) => {
  if (!html) return ''
  const parser = new DOMParser()
  const doc = parser.parseFromString(html, 'text/html')
  const tokens = doc.querySelectorAll('.ql-variable-token')
  tokens.forEach((el) => {
    const tag = el.getAttribute('data-variable') || el.textContent
    el.replaceWith(tag)
  })
  return doc.body.innerHTML
}

// --- Mail form ---
const mail = ref({
  sender: 0,
  subject: '',
  body: ''
})

const senders = ref([{ name: 'Nathejk <tilmeld@nathejk.dk>', code: 0 }])

const sending = ref(false)

const canSend = computed(() => {
  return selectedRecipients.value.length > 0 && mail.value.subject.trim().length > 0 && mail.value.body && mail.value.body.trim().length > 0
})

const sendMail = async () => {
  if (!canSend.value) return

  sending.value = true
  try {
    const payload = {
      sender: mail.value.sender,
      subject: mail.value.subject,
      body: serializeBody(mail.value.body),
      recipientType: selectedRecipientType.value,
      recipientScope: showScopeSelector.value ? selectedRecipientScope.value : 'direct',
      recipientIds: selectedRecipients.value.map((r) => r.id)
    }
    await http.post('/mail/send', payload)
    toast.add({
      severity: 'success',
      summary: 'E-mail sendt',
      detail: `Sendt til ${selectedRecipients.value.length} modtager(e)`,
      life: 4000
    })
    // Reset form
    mail.value.subject = ''
    mail.value.body = ''
    selectedPersons.value = []
    selectedPatruljer.value = []
    selectedKlaner.value = []
  } catch (error) {
    console.error('Failed to send mail', error)
    toast.add({
      severity: 'error',
      summary: 'Fejl',
      detail: 'E-mailen kunne ikke sendes. Prøv igen.',
      life: 5000
    })
  } finally {
    sending.value = false
  }
}

// =============================================
// --- Templates (localStorage-backed) ---
// =============================================
const TEMPLATES_STORAGE_KEY = 'nathejk-mail-templates'

const loadTemplates = () => {
  try {
    const raw = localStorage.getItem(TEMPLATES_STORAGE_KEY)
    return raw ? JSON.parse(raw) : []
  } catch {
    return []
  }
}

const persistTemplates = () => {
  localStorage.setItem(TEMPLATES_STORAGE_KEY, JSON.stringify(templates.value))
}

const templates = ref(loadTemplates())

const templatesByRecipientType = computed(() => {
  const groups = {}
  for (const rt of recipientTypes.value) {
    groups[rt.value] = templates.value.filter((t) => t.recipientType === rt.value).sort((a, b) => (b.lastUsedAt || '').localeCompare(a.lastUsedAt || ''))
  }
  return groups
})

const hasAnyTemplates = computed(() => templates.value.length > 0)

const generateId = () => {
  return Date.now().toString(36) + Math.random().toString(36).substring(2, 9)
}

const createTemplate = (recipientType) => {
  const tpl = {
    id: generateId(),
    recipientType: recipientType || 'person',
    title: '',
    subject: '',
    body: '',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    lastUsedAt: null
  }
  templates.value.push(tpl)
  persistTemplates()
  openTemplateEditor(tpl)
}

const deleteTemplate = (id) => {
  templates.value = templates.value.filter((t) => t.id !== id)
  persistTemplates()
  if (editingTemplate.value && editingTemplate.value.id === id) {
    editingTemplate.value = null
  }
  toast.add({
    severity: 'info',
    summary: 'Skabelon slettet',
    life: 3000
  })
}

// --- Template editor state ---
const editingTemplate = ref(null)

const openTemplateEditor = (tpl) => {
  // Work on a shallow copy so we can cancel
  editingTemplate.value = { ...tpl }
  lastFocusedField.value = 'body'
  subjectCursorPos.value = null
}

const cancelTemplateEdit = () => {
  editingTemplate.value = null
}

const saveTemplate = () => {
  if (!editingTemplate.value) return
  const tpl = editingTemplate.value
  tpl.updatedAt = new Date().toISOString()

  // Serialize body HTML to normalize variable blots
  tpl.body = serializeBody(tpl.body)

  const idx = templates.value.findIndex((t) => t.id === tpl.id)
  if (idx >= 0) {
    templates.value[idx] = { ...tpl }
  } else {
    templates.value.push({ ...tpl })
  }
  persistTemplates()
  editingTemplate.value = null
  toast.add({
    severity: 'success',
    summary: 'Skabelon gemt',
    life: 3000
  })
}

const useTemplate = (tpl) => {
  // Mark last used
  const idx = templates.value.findIndex((t) => t.id === tpl.id)
  if (idx >= 0) {
    templates.value[idx].lastUsedAt = new Date().toISOString()
    persistTemplates()
  }
  // Load into compose form
  selectedRecipientType.value = tpl.recipientType
  mail.value.subject = tpl.subject
  mail.value.body = tpl.body
  router.push('/mail/new')
}

const formatDate = (iso) => {
  if (!iso) return '—'
  const d = new Date(iso)
  return d.toLocaleDateString('da-DK', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

// Track which create-dropdown is open
const showNewTemplateMenu = ref(false)

const createTemplateFromMenu = (recipientType) => {
  showNewTemplateMenu.value = false
  createTemplate(recipientType)
}

// --- Sidebar menu ---
const menuItems = ref([
  {
    label: 'Mails',
    items: [
      { label: 'Skriv ny', icon: PrimeIcons.PENCIL, route: '/mail/new' },
      { label: 'Udbakke', icon: PrimeIcons.ENVELOPE, route: '/mail/outbox' },
      { label: 'Skabeloner', icon: PrimeIcons.FOLDER_OPEN, route: '/mail/templates' }
    ]
  }
])
</script>

<template>
  <h1 class="font-nathejk text-2xl py-3">
    Kommunikation
    <span
      >: <span class="text-slate-400">{{ pageTitle }}</span></span
    >
  </h1>

  <div class="grid grid-cols-6 gap-4 mb-3">
    <!-- Sidebar -->
    <div class="card flex justify-center">
      <Menu :model="menuItems">
        <template #item="{ item, props }">
          <router-link v-if="item.route" v-slot="{ href, navigate }" :to="item.route" custom>
            <a v-ripple :href="href" v-bind="props.action" @click="navigate">
              <span :class="item.icon" />
              <span class="ml-2">{{ item.label }}</span>
            </a>
          </router-link>
        </template>
      </Menu>
    </div>

    <!-- ============================================= -->
    <!-- COMPOSE PAGE -->
    <!-- ============================================= -->
    <div v-if="currentPage === 'compose'" class="col-span-5">
      <div class="card">
        <div class="grid grid-cols-3 gap-4">
          <!-- Step 1: Recipient type -->
          <div class="col-span-3">
            <label class="block text-sm font-semibold text-slate-600 mb-2"> <i class="pi pi-users mr-1"></i>Modtagertype </label>
            <div class="flex gap-2">
              <Button v-for="rt in recipientTypes" :key="rt.value" :label="rt.label" :icon="rt.icon" :outlined="selectedRecipientType !== rt.value" :severity="selectedRecipientType === rt.value ? undefined : 'secondary'" size="small" @click="selectedRecipientType = rt.value" />
            </div>
          </div>

          <!-- Step 2: Pick recipients -->
          <div class="col-span-2">
            <!-- Person picker -->
            <div v-if="selectedRecipientType === 'person'">
              <FloatLabel variant="on">
                <MultiSelect v-model="selectedPersons" :options="personnel" optionLabel="name" filter :maxSelectedLabels="5" :loading="loading" class="w-full" display="chip" id="person-select" fluid>
                  <template #option="{ option }">
                    <div class="flex items-center gap-2">
                      <i class="pi pi-user text-slate-400"></i>
                      <div>
                        <div class="font-medium">{{ option.name }}</div>
                        <div class="text-xs text-slate-500" v-if="option.group">{{ option.group }}</div>
                      </div>
                    </div>
                  </template>
                </MultiSelect>
                <label for="person-select">Vælg person(er)</label>
              </FloatLabel>
            </div>

            <!-- Patrulje picker -->
            <div v-if="selectedRecipientType === 'patrulje'">
              <FloatLabel variant="on">
                <MultiSelect v-model="selectedPatruljer" :options="patruljer" optionLabel="name" filter :maxSelectedLabels="5" :loading="loading" class="w-full" display="chip" id="patrulje-select" fluid>
                  <template #option="{ option }">
                    <div class="flex items-center gap-2">
                      <i class="pi pi-users text-slate-400"></i>
                      <div>
                        <div class="font-medium">
                          <span v-if="option.teamNumber" class="text-slate-500 mr-1">#{{ option.teamNumber }}</span>
                          {{ option.name }}
                        </div>
                        <div class="text-xs text-slate-500">
                          {{ option.group }}
                          <span v-if="option.memberCount"> · {{ option.memberCount }} spejdere</span>
                        </div>
                      </div>
                    </div>
                  </template>
                </MultiSelect>
                <label for="patrulje-select">Vælg patrulje(r)</label>
              </FloatLabel>
            </div>

            <!-- Klan picker -->
            <div v-if="selectedRecipientType === 'klan'">
              <FloatLabel variant="on">
                <MultiSelect v-model="selectedKlaner" :options="klaner" optionLabel="name" filter :maxSelectedLabels="5" :loading="loading" class="w-full" display="chip" id="klan-select" fluid>
                  <template #option="{ option }">
                    <div class="flex items-center gap-2">
                      <i class="pi pi-flag text-slate-400"></i>
                      <div>
                        <div class="font-medium">{{ option.name }}</div>
                        <div class="text-xs text-slate-500" v-if="option.group">
                          {{ option.group }}
                          <span v-if="option.memberCount"> · {{ option.memberCount }} medlemmer</span>
                        </div>
                      </div>
                    </div>
                  </template>
                </MultiSelect>
                <label for="klan-select">Vælg klan(er)</label>
              </FloatLabel>
            </div>
          </div>

          <!-- Recipient scope (only for patrulje/klan) -->
          <div class="col-span-1 flex items-center">
            <div v-if="showScopeSelector">
              <label class="block text-xs font-semibold text-slate-500 mb-1">Hvem modtager?</label>
              <div class="flex flex-col gap-1">
                <div v-for="scope in recipientScopes" :key="scope.value" class="flex items-center gap-2">
                  <RadioButton v-model="selectedRecipientScope" :inputId="'scope-' + scope.value" :value="scope.value" />
                  <label :for="'scope-' + scope.value" class="text-sm cursor-pointer">
                    {{ scope.label }}
                  </label>
                </div>
              </div>
            </div>
          </div>

          <!-- Recipient summary badge -->
          <div class="col-span-3" v-if="selectedRecipients.length > 0">
            <div class="flex items-center gap-2 px-3 py-2 bg-blue-50 rounded-lg border border-blue-200 text-sm text-blue-800">
              <i class="pi pi-info-circle"></i>
              <span>
                <strong>{{ selectedRecipients.length }}</strong>
                {{ selectedRecipientType === 'person' ? 'person(er)' : selectedRecipientType === 'patrulje' ? 'patrulje(r)' : 'klan(er)' }}
                valgt
                <span v-if="showScopeSelector">
                  — sender til <strong>{{ selectedRecipientScope === 'contact' ? 'kontaktpersoner' : 'alle medlemmer' }}</strong>
                </span>
              </span>
            </div>
          </div>

          <Divider class="col-span-3 my-0" />

          <!-- Sender -->
          <div class="col-span-2">
            <FloatLabel variant="on">
              <Select v-model="mail.sender" id="sender" :options="senders" optionLabel="name" optionValue="code" class="w-full" fluid />
              <label for="sender">Afsender</label>
            </FloatLabel>
          </div>

          <!-- Subject -->
          <div class="col-span-2">
            <FloatLabel variant="on">
              <InputText ref="subjectRef" type="text" v-model="mail.subject" id="subject" class="w-full" fluid @focus="onSubjectFocus" @blur="onSubjectBlur" />
              <label for="subject">Emne</label>
            </FloatLabel>
          </div>

          <!-- Editor + variables side panel -->
          <div class="col-span-2">
            <Editor ref="editorRef" v-model="mail.body" editorStyle="height: 320px" fluid @selection-change="onEditorFocus">
              <template v-slot:toolbar>
                <span class="ql-formats">
                  <button v-tooltip.bottom="'Fed'" class="ql-bold"></button>
                  <button v-tooltip.bottom="'Kursiv'" class="ql-italic"></button>
                  <button v-tooltip.bottom="'Understregning'" class="ql-underline"></button>
                </span>
                <span class="ql-formats">
                  <button v-tooltip.bottom="'Liste'" class="ql-list" value="bullet"></button>
                  <button v-tooltip.bottom="'Nummereret liste'" class="ql-list" value="ordered"></button>
                </span>
                <span class="ql-formats">
                  <button v-tooltip.bottom="'Link'" class="ql-link"></button>
                </span>
              </template>
            </Editor>
          </div>

          <!-- Variable panel -->
          <div class="col-span-1">
            <div class="border border-slate-200 rounded-lg p-3 bg-slate-50 h-full">
              <h3 class="text-sm font-semibold text-slate-600 mb-2 flex items-center gap-1">
                <i class="pi pi-code text-xs"></i>
                Variabler
              </h3>
              <p class="text-xs text-slate-400 mb-3">Klik for at indsætte i emne eller brødtekst</p>
              <div class="flex flex-col gap-1">
                <button v-for="v in availableVariables" :key="v.tag" class="text-left px-2 py-1.5 rounded text-sm hover:bg-blue-100 hover:text-blue-800 transition-colors group cursor-pointer border border-transparent hover:border-blue-200" @click="insertVariable(v.tag)" v-tooltip.left="v.description">
                  <span class="font-mono text-xs text-blue-600 group-hover:text-blue-800">{{ v.tag }}</span>
                  <span class="block text-xs text-slate-500 group-hover:text-blue-600">{{ v.label }}</span>
                </button>
              </div>

              <Divider />

              <h4 class="text-xs font-semibold text-slate-500 mb-1">Generelle</h4>
              <div class="flex flex-col gap-1">
                <button class="text-left px-2 py-1.5 rounded text-sm hover:bg-blue-100 hover:text-blue-800 transition-colors group cursor-pointer border border-transparent hover:border-blue-200" @click="insertVariable('#ÅRSTAL#')" v-tooltip.left="'Indeværende årstal'">
                  <span class="font-mono text-xs text-blue-600 group-hover:text-blue-800">#ÅRSTAL#</span>
                  <span class="block text-xs text-slate-500 group-hover:text-blue-600">Årstal</span>
                </button>
                <button class="text-left px-2 py-1.5 rounded text-sm hover:bg-blue-100 hover:text-blue-800 transition-colors group cursor-pointer border border-transparent hover:border-blue-200" @click="insertVariable('#AFSENDER#')" v-tooltip.left="'Afsenderens navn'">
                  <span class="font-mono text-xs text-blue-600 group-hover:text-blue-800">#AFSENDER#</span>
                  <span class="block text-xs text-slate-500 group-hover:text-blue-600">Afsender</span>
                </button>
              </div>
            </div>
          </div>

          <!-- Send button row -->
          <div class="col-span-3 flex items-center justify-between pt-2">
            <span class="text-sm text-slate-400">
              {{ recipientSummary }}
            </span>
            <Button :label="sending ? 'Sender...' : 'Send e-mail'" icon="pi pi-send" :loading="sending" :disabled="!canSend" @click="sendMail" />
          </div>
        </div>
      </div>
    </div>

    <!-- ============================================= -->
    <!-- TEMPLATES PAGE -->
    <!-- ============================================= -->
    <div v-if="currentPage === 'templates'" class="col-span-5">
      <!-- Template editor (editing a specific template) -->
      <div v-if="editingTemplate" class="card">
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-2">
            <button class="text-slate-400 hover:text-slate-600 transition-colors cursor-pointer" @click="cancelTemplateEdit" v-tooltip.top="'Tilbage til oversigt'">
              <i class="pi pi-arrow-left text-lg"></i>
            </button>
            <h2 class="text-lg font-semibold text-slate-700">Redigér skabelon</h2>
          </div>
          <div class="flex items-center gap-1">
            <span class="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium bg-slate-100 text-slate-600">
              <i :class="recipientTypeIcon(editingTemplate.recipientType)" class="text-xs"></i>
              {{ recipientTypeLabel(editingTemplate.recipientType) }}
            </span>
          </div>
        </div>

        <div class="grid grid-cols-3 gap-4">
          <!-- Title -->
          <div class="col-span-3">
            <FloatLabel variant="on">
              <InputText type="text" v-model="editingTemplate.title" id="tpl-title" class="w-full" fluid />
              <label for="tpl-title">Skabelonnavn</label>
            </FloatLabel>
          </div>

          <!-- Recipient type selector -->
          <div class="col-span-3">
            <label class="block text-sm font-semibold text-slate-600 mb-2"> <i class="pi pi-users mr-1"></i>Modtagertype </label>
            <div class="flex gap-2">
              <Button v-for="rt in recipientTypes" :key="rt.value" :label="rt.label" :icon="rt.icon" :outlined="editingTemplate.recipientType !== rt.value" :severity="editingTemplate.recipientType === rt.value ? undefined : 'secondary'" size="small" @click="editingTemplate.recipientType = rt.value" />
            </div>
          </div>

          <!-- Subject -->
          <div class="col-span-2">
            <FloatLabel variant="on">
              <InputText ref="subjectRef" type="text" v-model="editingTemplate.subject" id="tpl-subject" class="w-full" fluid @focus="onSubjectFocus" @blur="onSubjectBlur" />
              <label for="tpl-subject">Emne</label>
            </FloatLabel>
          </div>

          <!-- Editor -->
          <div class="col-span-2">
            <Editor ref="editorRef" v-model="editingTemplate.body" editorStyle="height: 320px" fluid @selection-change="onEditorFocus">
              <template v-slot:toolbar>
                <span class="ql-formats">
                  <button v-tooltip.bottom="'Fed'" class="ql-bold"></button>
                  <button v-tooltip.bottom="'Kursiv'" class="ql-italic"></button>
                  <button v-tooltip.bottom="'Understregning'" class="ql-underline"></button>
                </span>
                <span class="ql-formats">
                  <button v-tooltip.bottom="'Liste'" class="ql-list" value="bullet"></button>
                  <button v-tooltip.bottom="'Nummereret liste'" class="ql-list" value="ordered"></button>
                </span>
                <span class="ql-formats">
                  <button v-tooltip.bottom="'Link'" class="ql-link"></button>
                </span>
              </template>
            </Editor>
          </div>

          <!-- Variable panel -->
          <div class="col-span-1">
            <div class="border border-slate-200 rounded-lg p-3 bg-slate-50 h-full">
              <h3 class="text-sm font-semibold text-slate-600 mb-2 flex items-center gap-1">
                <i class="pi pi-code text-xs"></i>
                Variabler
              </h3>
              <p class="text-xs text-slate-400 mb-3">Klik for at indsætte i emne eller brødtekst</p>
              <div class="flex flex-col gap-1">
                <button v-for="v in availableVariables" :key="v.tag" class="text-left px-2 py-1.5 rounded text-sm hover:bg-blue-100 hover:text-blue-800 transition-colors group cursor-pointer border border-transparent hover:border-blue-200" @click="insertVariable(v.tag)" v-tooltip.left="v.description">
                  <span class="font-mono text-xs text-blue-600 group-hover:text-blue-800">{{ v.tag }}</span>
                  <span class="block text-xs text-slate-500 group-hover:text-blue-600">{{ v.label }}</span>
                </button>
              </div>

              <Divider />

              <h4 class="text-xs font-semibold text-slate-500 mb-1">Generelle</h4>
              <div class="flex flex-col gap-1">
                <button class="text-left px-2 py-1.5 rounded text-sm hover:bg-blue-100 hover:text-blue-800 transition-colors group cursor-pointer border border-transparent hover:border-blue-200" @click="insertVariable('#ÅRSTAL#')" v-tooltip.left="'Indeværende årstal'">
                  <span class="font-mono text-xs text-blue-600 group-hover:text-blue-800">#ÅRSTAL#</span>
                  <span class="block text-xs text-slate-500 group-hover:text-blue-600">Årstal</span>
                </button>
                <button class="text-left px-2 py-1.5 rounded text-sm hover:bg-blue-100 hover:text-blue-800 transition-colors group cursor-pointer border border-transparent hover:border-blue-200" @click="insertVariable('#AFSENDER#')" v-tooltip.left="'Afsenderens navn'">
                  <span class="font-mono text-xs text-blue-600 group-hover:text-blue-800">#AFSENDER#</span>
                  <span class="block text-xs text-slate-500 group-hover:text-blue-600">Afsender</span>
                </button>
              </div>
            </div>
          </div>

          <!-- Save / Cancel row -->
          <div class="col-span-3 flex items-center justify-between pt-2 border-t border-slate-100">
            <Button label="Annullér" icon="pi pi-times" severity="secondary" text @click="cancelTemplateEdit" />
            <div class="flex gap-2">
              <Button label="Brug skabelon" icon="pi pi-file-export" severity="secondary" outlined @click="useTemplate(editingTemplate)" />
              <Button label="Gem skabelon" icon="pi pi-save" @click="saveTemplate" :disabled="!editingTemplate.title?.trim()" />
            </div>
          </div>
        </div>
      </div>

      <!-- Template list (overview) -->
      <div v-else>
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-lg font-semibold text-slate-700">Mailskabeloner</h2>
          <div class="relative">
            <Button label="Ny skabelon" icon="pi pi-plus" size="small" @click="showNewTemplateMenu = !showNewTemplateMenu" />
            <div v-if="showNewTemplateMenu" class="absolute right-0 top-full mt-1 bg-white border border-slate-200 rounded-lg shadow-lg z-10 py-1 min-w-48">
              <button v-for="rt in recipientTypes" :key="rt.value" class="w-full text-left px-4 py-2 text-sm hover:bg-slate-50 flex items-center gap-2 cursor-pointer transition-colors" @click="createTemplateFromMenu(rt.value)">
                <i :class="rt.icon" class="text-slate-400"></i>
                <span>{{ rt.label }}</span>
              </button>
            </div>
          </div>
        </div>

        <!-- Empty state -->
        <div v-if="!hasAnyTemplates" class="card text-center py-12">
          <i class="pi pi-folder-open text-4xl text-slate-300 mb-3 block"></i>
          <h3 class="text-lg font-medium text-slate-500 mb-1">Ingen skabeloner endnu</h3>
          <p class="text-sm text-slate-400 mb-4">Opret en skabelon for at genbruge emne og brødtekst</p>
          <div class="flex justify-center gap-2">
            <Button v-for="rt in recipientTypes" :key="rt.value" :label="rt.label" :icon="rt.icon" size="small" severity="secondary" outlined @click="createTemplate(rt.value)" />
          </div>
        </div>

        <!-- Template groups by recipient type -->
        <div v-else class="flex flex-col gap-6">
          <div v-for="rt in recipientTypes" :key="rt.value">
            <div v-if="templatesByRecipientType[rt.value]?.length > 0">
              <div class="flex items-center gap-2 mb-3">
                <i :class="rt.icon" class="text-slate-400"></i>
                <h3 class="text-sm font-semibold text-slate-600 uppercase tracking-wide">{{ rt.label }}</h3>
                <span class="text-xs text-slate-400">({{ templatesByRecipientType[rt.value].length }})</span>
              </div>

              <div class="grid grid-cols-1 gap-2">
                <div v-for="tpl in templatesByRecipientType[rt.value]" :key="tpl.id" class="card border border-slate-200 hover:border-blue-300 hover:shadow-sm transition-all cursor-pointer group" @click="openTemplateEditor(tpl)">
                  <div class="flex items-start justify-between">
                    <div class="flex-1 min-w-0">
                      <div class="flex items-center gap-2 mb-1">
                        <h4 class="font-medium text-slate-700 truncate">{{ tpl.title || 'Unavngivet skabelon' }}</h4>
                      </div>
                      <div v-if="tpl.subject" class="text-sm text-slate-500 truncate mb-1"><i class="pi pi-envelope text-xs mr-1"></i>{{ tpl.subject }}</div>
                      <div class="flex items-center gap-4 text-xs text-slate-400">
                        <span v-if="tpl.updatedAt"> <i class="pi pi-pencil mr-1"></i>Redigeret {{ formatDate(tpl.updatedAt) }} </span>
                        <span v-if="tpl.lastUsedAt"> <i class="pi pi-send mr-1"></i>Sidst brugt {{ formatDate(tpl.lastUsedAt) }} </span>
                        <span v-else class="italic">Aldrig brugt</span>
                      </div>
                    </div>
                    <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity ml-3">
                      <Button icon="pi pi-file-export" v-tooltip.top="'Brug skabelon'" size="small" severity="secondary" text rounded @click.stop="useTemplate(tpl)" />
                      <Button icon="pi pi-pencil" v-tooltip.top="'Redigér'" size="small" severity="secondary" text rounded @click.stop="openTemplateEditor(tpl)" />
                      <Button icon="pi pi-trash" v-tooltip.top="'Slet'" size="small" severity="danger" text rounded @click.stop="deleteTemplate(tpl.id)" />
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ============================================= -->
    <!-- OUTBOX PAGE (placeholder) -->
    <!-- ============================================= -->
    <div v-if="currentPage === 'outbox'" class="col-span-5">
      <div class="card text-center py-12">
        <i class="pi pi-envelope text-4xl text-slate-300 mb-3 block"></i>
        <h3 class="text-lg font-medium text-slate-500">Udbakke</h3>
        <p class="text-sm text-slate-400">Kommer snart...</p>
      </div>
    </div>
  </div>
</template>

<style>
/* Variable token styling inside Quill editor */
.ql-variable-token {
  display: inline-flex;
  align-items: center;
  background: #dbeafe;
  color: #1e40af;
  border: 1px solid #93c5fd;
  border-radius: 4px;
  padding: 1px 6px;
  margin: 0 1px;
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, monospace;
  font-size: 0.8em;
  font-weight: 600;
  line-height: 1.6;
  white-space: nowrap;
  cursor: default;
  user-select: all;
  vertical-align: baseline;
}

.ql-variable-token::before {
  content: '';
  display: inline-block;
  width: 6px;
  height: 6px;
  background: #3b82f6;
  border-radius: 50%;
  margin-right: 4px;
  flex-shrink: 0;
}

.ql-variable-token:hover {
  background: #bfdbfe;
  border-color: #60a5fa;
}
</style>
